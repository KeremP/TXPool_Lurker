package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	pendingTxNotFoundMaxRetries = 2
	pendingTxNotFoundDelay      = 100 * time.Millisecond
)

type (
	PendingTx struct {
		resolver pendingTxResolver
		handler  pendingTxHandler
	}

	pendingTxResolver interface {
		TransactionByHash(context.Context, common.Hash) (tx *types.Transaction, isPending bool, err error)
	}

	pendingTxHandler func(context.Context, *types.Transaction, bool) error

	pendingTxNotFoundKey struct{}
)

func NewPendingTx(resolver pendingTxResolver, handler pendingTxHandler) *PendingTx {
	return &PendingTx{
		resolver: resolver,
		handler:  handler,
	}
}

func (c *PendingTx) Snipe(ctx context.Context, h common.Hash) error {
	log.Trace(fmt.Sprintf("new tx: %s", h.Hex()))

	tx, pending, err := c.resolver.TransactionByHash(ctx, h)

	if err == ethereum.NotFound {
		_time, newCtx := c.newContextForRetries(ctx)
		if _time < pendingTxNotFoundMaxRetries {
			log.Debug(fmt.Sprintf("tx not found. retrying: %s", h.Hex()))
			time.AfterFunc(pendingTxNotFoundDelay, func() {
				if err := c.Snipe(newCtx, h); err != nil {
					log.Error(err.Error())
				}
			})
		} else {
			log.Warn(fmt.Sprintf("tx not found: %s", h.Hex()))
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("error retrieving tx %s by hash: %s", h.Hex(), err)
	}

	if pending {
		return c.handler(ctx, tx, pending)
	}
	log.Warn(fmt.Sprintf("tx already confirmed: %s", h.Hex()))
	return nil
}

func (c *PendingTx) newContextForRetries(ctx context.Context) (uint, context.Context) {
	var times uint = 0

	if v, ok := ctx.Value(pendingTxNotFoundKey{}).(uint); ok {
		times = v
	}
	return times, context.WithValue(ctx, pendingTxNotFoundKey{}, times+1)
}
