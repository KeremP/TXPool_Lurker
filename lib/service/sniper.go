package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/keremp/TXPool_Lurker/lib/domain"
	erc20 "github.com/keremp/TXPool_Lurker/third_party"
)

const (
	nullHash = "0x0000000000000000000000000000000000000000000000000000000000000000"

	sniperMaxWaitTimeForTx = 20 * time.Second
)

var (
	triggerSmartContract = []byte{0x4e, 0xfa, 0xc3, 0x29}
	txValue              = big.NewInt(0)
	txGasLimit           = uint64(500000)
)

type (
	Sniper struct {
		mut *sync.Mutex

		factoryClient sniperFactoryClient
		ethClient     sniperEthClient
		swarm         []*Bee

		sniperTTBAddr     common.Address
		sniperTriggerAddr common.Address
		sniperTokenPaired common.Address
		sniperChainId     *big.Int
	}

	sniperFactoryClient interface {
		GetPair(opts *bind.CallOpts, tokenA common.Address, tokenB common.Address) (common.Address, error)
	}

	sniperEthClient interface {
		bind.ContractBackend

		SendTransaction(context.Context, *types.Transaction) error

		TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)

		TransactionByReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	}

	Bee struct {
		RawPk        *ecdsa.PrivateKey
		PendingNonce uint64
	}

	txResp struct {
		Hash    common.Hash
		Receipt *types.Receipt
		Success bool
	}
)

func NewSniper(e sniperEthClient, f sniperFactoryClient, s []*Bee, sn domain.Sniper) *Sniper {
	return &Sniper{
		mut:               new(sync.Mutex),
		ethClient:         e,
		factoryClient:     f,
		swarm:             s,
		sniperTTBAddr:     common.HexToAddress(sn.AddressTargetToken),
		sniperTriggerAddr: common.HexToAddress(sn.AddressTrigger),
		sniperTokenPaired: common.HexToAddress(sn.AddressTargetPaired),
		sniperChainId:     sn.ChainID,
	}
}

func NewBee(rawPk *ecdsa.PrivateKey, pn uint64) *Bee {
	return &Bee{
		RawPk:        rawPk,
		PendingNonce: pn,
	}
}

// snipe function
func (c *Sniper) Snipe(ctx context.Context, gas *big.Int) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	wg := new(sync.WaitGroup)
	wg.Add(len(c.swarm))

	pendingTxResp := make(chan common.Hash, len(c.swarm))

	for _, b := range c.swarm {
		go func(ctx context.Context, b *Bee, wg *sync.WaitGroup, gas *big.Int, h chan<- common.Hash) {
			defer recovery()
			defer wg.Done()
			h <- c.execute(ctx, b, gas)
		}(ctx, b, wg, gas, pendingTxResp)
	}

	wg.Wait()
	close(pendingTxResp)

	finishedTxResp := make(chan txResp, len(pendingTxResp))
	wg.Add(len(pendingTxResp))

	for txHash := range pendingTxResp {
		go func(ctx context.Context, h common.Hash, wg *sync.WaitGroup, ch chan<- txResp) {
			defer recovery()
			defer wg.Done()
			ch <- c.checkTxStatus(ctx, h)
		}(ctx, txHash, wg, finishedTxResp)
	}

	wg.Wait()
	close(finishedTxResp)

	success := false
	for resp := range finishedTxResp {
		if resp.Success {
			success = true
			for _, l := range resp.Receipt.Logs {
				if l.Address == c.sniperTTBAddr {
					hexAmount := hex.EncodeToString(l.Data)
					var value = new(big.Int)
					value.SetString(hexAmount, 16)
					var buf strings.Builder
					_, _ = buf.WriteString("Sniping succeeded\n")
					_, _ = buf.WriteString(fmt.Sprintf(" Hash: %s\n", resp.Hash.String()))
					_, _ = buf.WriteString(fmt.Sprintf(" Token: %s\n", c.sniperTTBAddr.String()))

					if amountBought, err := c.formatER20Decimals(value, c.sniperTTBAddr); err == nil {
						_, _ = buf.WriteString(fmt.Sprintf(" Amount purchased: %f", amountBought))
					}

					if pairAddress, err := c.factoryClient.GetPair(&bind.CallOpts{}, c.sniperTTBAddr, c.sniperTokenPaired); err == nil {
						_, _ = buf.WriteString(fmt.Sprintf(" Pair Address: %s", pairAddress.String()))
					}

					log.Info(buf.String())
				}
			}
		}
	}
	if !success {
		log.Warn("Sniping unsuccessful.")
	}
	return nil
}

// Format number of token transferred
func (c *Sniper) formatER20Decimals(tokensSent *big.Int, tokenAddress common.Address) (float64, error) {
	tokenInstance, _ := erc20.NewErc20(tokenAddress, c.ethClient)
	decimals, err := tokenInstance.Decimals(nil)
	if err != nil {
		return 0, err
	}

	var base, exponent = big.NewInt(10), big.NewInt(int64(decimals))
	denominator := base.Exp(base, exponent, nil)

	tokensSentFloat := new(big.Float).SetInt(tokensSent)
	denominatorFloat := new(big.Float).SetInt(denominator)

	final, _ := new(big.Float).Quo(tokensSentFloat, denominatorFloat).Float64()
	return final, nil
}

func (c *Sniper) checkTxStatus(ctx context.Context, txHash common.Hash) txResp {
	if txHash == common.HexToHash(nullHash) {
		return txResp{
			Hash:    txHash,
			Success: false,
			Receipt: nil,
		}
	}

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

	s := time.Now()

	for range t.C {
		_, pend, err := c.ethClient.TransactionByHash(ctx, txHash)

		if !pend {
			break
		}

		if err != nil {
			log.Error(fmt.Sprintf("Error fetching transaction by hash %s: %s", txHash.String(), err))
		}

		if time.Now().Add(-sniperMaxWaitTimeForTx).After(s) {
			return txResp{
				Hash:    txHash,
				Success: false, //true
				Receipt: nil,
			}
		}
	}

	receipt, err := c.ethClient.TransactionByReceipt(ctx, txHash)

	if err != nil {
		log.Error(fmt.Sprintf("Error fetching transaction receipt %s: %s", txHash.String(), err.Error()))
		return txResp{
			Hash:    txHash,
			Success: false,
			Receipt: nil,
		}
	}

	return txResp{
		Hash:    txHash,
		Success: receipt.Status == 1,
		Receipt: receipt,
	}

}

func (c *Sniper) execute(ctx context.Context, bee *Bee, gasPrice *big.Int) common.Hash {
	log.Debug(fmt.Sprintf("Gas price: %s", gasPrice.String()))
	nonce := bee.PendingNonce

	// build tx
	txBee := types.NewTransaction(nonce, c.sniperTriggerAddr, txValue, txGasLimit, gasPrice, triggerSmartContract)

	signedTxBee, err := types.SignTx(txBee, types.LatestSignerForChainID(c.sniperChainId), bee.RawPk)

	if err != nil {
		log.Error(fmt.Sprintf("Issue with signed Bee Tx: %s", err))
		return common.HexToHash(nullHash)
	}

	err = c.ethClient.SendTransaction(ctx, signedTxBee)

	if err != nil {
		log.Error(fmt.Sprintf("error sending tx: %s", err.Error()))
		return common.HexToHash(nullHash)
	}

	log.Info(fmt.Sprintf("sent tx: %s", signedTxBee.Hash().Hex()))
	bee.PendingNonce++

	return signedTxBee.Hash()
}

func recovery() {
	if err := recover(); err != nil {
		log.Error(fmt.Sprintf("panic recovered: %s %s", fmt.Errorf("%s", err), debug.Stack()))
	}
}
