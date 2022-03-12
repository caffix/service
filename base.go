// Copyright © by Jeff Foley 2020-2022. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"errors"
	"sync"

	"go.uber.org/ratelimit"
)

// BaseService provides common mechanisms to all services implementing the Service interface.
type BaseService struct {
	sync.Mutex
	name   string
	runs   bool
	done   chan struct{}
	input  chan interface{}
	output chan interface{}
	rlock  sync.Mutex
	rlimit ratelimit.Limiter
	// The specific service embedding BaseService
	service Service
}

// NewBaseService returns an initialized BaseService object.
func NewBaseService(srv Service, name string) *BaseService {
	return &BaseService{
		name:    name,
		done:    make(chan struct{}),
		input:   make(chan interface{}),
		output:  make(chan interface{}),
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

// Done implements the Service interface.
func (bas *BaseService) Done() <-chan struct{} {
	return bas.done
}

// Input implements the Service interface.
func (bas *BaseService) Input() chan interface{} {
	return bas.input
}

// Output implements the Service interface.
func (bas *BaseService) Output() chan interface{} {
	return bas.output
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
