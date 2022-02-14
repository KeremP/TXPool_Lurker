package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type (
	Engine struct {
		client *rpc.Client
		sub    engineSub
		middle engineMid
		ctrl   engineCtrl
	}

	engineSub  func(ctx context.Context, c *rpc.Client, ch chan<- interface{}) (*rpc.ClientSubscription, error)
	engineMid  func(context.Context) context.Context
	engineCtrl func(ctx context.Context, v interface{}) error
)

func NewEngine(cl *rpc.Client, sub engineSub, mid engineMid, ctrl engineCtrl) *Engine {
	return &Engine{
		client: cl,
		sub:    sub,
		middle: mid,
		ctrl:   ctrl,
	}
}

func (e *Engine) Run(ctx context.Context, workers int) {
	var canc func()

	ctx, canc = context.WithCancel(ctx)

	ch := make(chan interface{}, workers)

	wg := new(sync.WaitGroup)
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(ctx context.Context, ch <-chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			defer recovery(canc)

			e.consumeBlocking(ctx, ch)
		}(ctx, ch, wg)
	}

	e.subscribeSafe(ctx, ch, canc)

	wg.Wait()
}

func (e *Engine) subscribeSafe(ctx context.Context, ch chan<- interface{}, canc func()) {
	s, err := e.sub(ctx, e.client, ch)
	if err != nil {
		panic(err)
	}
	go func() {
		defer recovery(canc)
		var once sync.Once

		for range s.Err() {
			once.Do(func() {
				s.Unsubscribe()
				e.subscribeSafe(ctx, ch, canc)
			})
		}
	}()
}

func (e *Engine) consumeBlocking(ctx context.Context, ch <-chan interface{}) {
	for {
		select {
		case v := <-ch:
			if err := e.ctrl(e.middle(ctx), v); err != nil {
				log.Error(err.Error())
			}
		case <-ctx.Done():
			return
		}
	}
}

func recovery(fn func()) {
	if err := recover(); err != nil {
		log.Error(fmt.Sprintf("panic recovered: %s %s", fmt.Errorf("%s", err), debug.Stack()))
		if fn != nil {
			fn()
		}
	}
}
