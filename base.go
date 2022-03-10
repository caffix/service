// Copyright © by Jeff Foley 2020-2022. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/caffix/queue"
	"go.uber.org/ratelimit"
)

// BaseService provides common mechanisms to all services implementing the Service interface.
type BaseService struct {
	sync.Mutex
	name   string
	runs   bool
	queue  queue.Queue
	done   chan struct{}
	rlock  sync.Mutex
	rlimit ratelimit.Limiter
	// The specific service embedding BaseService
	service Service
}

// NewBaseService returns an initialized BaseService object.
func NewBaseService(srv Service, name string) *BaseService {
	return &BaseService{
		name:    name,
		queue:   queue.NewQueue(),
		service: srv,
	}
}

// Description implements the Service interface.
func (bas *BaseService) Description() string {
	return ""
}

// Start implements the Service interface.
func (bas *BaseService) Start() error {
	if bas.running() {
		return errors.New(bas.name + " has already been started")
	}

	bas.done = make(chan struct{})
	go bas.processRequests()
	bas.setRunning(true)
	return bas.service.OnStart()
}

// OnStart implements the Service interface.
func (bas *BaseService) OnStart() error {
	return nil
}

func (bas *BaseService) running() bool {
	bas.Lock()
	defer bas.Unlock()

	return bas.runs
}

func (bas *BaseService) setRunning(val bool) {
	bas.Lock()
	defer bas.Unlock()

	bas.runs = val
}

// Stop implements the Service interface.
func (bas *BaseService) Stop() error {
	if !bas.running() {
		return errors.New(bas.name + " is already stopped")
	}

	bas.setRunning(false)
	close(bas.done)
	return bas.service.OnStop()
}

// OnStop implements the Service interface.
func (bas *BaseService) OnStop() error {
	return nil
}

// Len implements the Service interface.
func (bas *BaseService) Len() int {
	return bas.queue.Len()
}

// Request implements the Service interface.
func (bas *BaseService) Request(ctx context.Context, args Args) {
	if bas.running() {
		bas.queueRequest(bas.service.OnRequest, ctx, args)
	}
}

// OnRequest implements the Service interface.
func (bas *BaseService) OnRequest(ctx context.Context, args Args) {}

// Done implements the Service interface.
func (bas *BaseService) Done() <-chan struct{} {
	return bas.done
}

// String implements the Stringer interface.
func (bas *BaseService) String() string {
	return bas.name
}

// SetRateLimit implements the Service interface.
func (bas *BaseService) SetRateLimit(persec int) {
	bas.rlock.Lock()
	defer bas.rlock.Unlock()

	if persec == 0 {
		bas.rlimit = nil
		return
	}

	bas.rlimit = ratelimit.New(persec, ratelimit.WithoutSlack)
}

// CheckRateLimit implements the Service interface.
func (bas *BaseService) CheckRateLimit() {
	bas.rlock.Lock()
	rlimit := bas.rlimit
	bas.rlock.Unlock()

	if rlimit != nil {
		rlimit.Take()
	}
}

type queuedCall struct {
	Func reflect.Value
	Args []reflect.Value
}

func (bas *BaseService) queueRequest(fn interface{}, args ...Args) {
	passedArgs := make([]reflect.Value, 0)
	for _, arg := range args {
		passedArgs = append(passedArgs, reflect.ValueOf(arg))
	}

	bas.queue.Append(&queuedCall{
		Func: reflect.ValueOf(fn),
		Args: passedArgs,
	})
}

func (bas *BaseService) processRequests() {
	each := func(element interface{}) {
		e := element.(*queuedCall)
		ctx := e.Args[0].Interface().(context.Context)

		select {
		case <-ctx.Done():
		case <-bas.Done():
		default:
			e.Func.Call(e.Args)
		}
	}

	for {
		select {
		case <-bas.Done():
			return
		case <-bas.queue.Signal():
			bas.queue.Process(each)
		}
	}
}
