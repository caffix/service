// Copyright 2020 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package service

import (
	"context"
	"fmt"
)

// Args is the type used to pass arguments to Service requests.
type Args interface{}

// Service handles queued requests at an optional rate limit.
type Service interface {
	fmt.Stringer

	// Description returns a greeting message from the service.
	Description() string

	// Start requests that the service be started.
	Start() error

	// OnStart is called when the Start method requests the service be started.
	OnStart() error

	// Stop requests that the service be stopped.
	Stop() error

	// OnStop is called when the Stop method requests the service be stopped.
	OnStop() error

	// Request queues a request for the service.
	Request(ctx context.Context, args Args)

	// OnRequest is called non-concurrently to handle a request appended to the queue.
	OnRequest(ctx context.Context, args Args)

	// Len returns the current length of the request queue.
	Len() int

	// Done returns a channel that is closed when the service is stopped.
	Done() <-chan struct{}

	// SetRateLimit sets the number of calls to the OnRequest method each second.
	SetRateLimit(persec int)

	// CheckRateLimit blocks until the minimum wait duration since the last call.
	CheckRateLimit()
}
