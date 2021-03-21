// Copyright 2020 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package service

import (
	"context"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	data := "testData"
	ch := make(chan string)
	srv := newTestService(ch)

	if srv.started {
		t.Errorf("The service started before calling Start")
	}

	srv.Request(context.TODO(), data)
	select {
	case <-ch:
		t.Errorf("The service called OnRequest before the Start method was called")
	default:
	}

	srv.Start()
	defer srv.Stop()

	if !srv.started {
		t.Errorf("The service did not start when the Start method was executed")
	}

	srv.Request(context.TODO(), data)
	time.Sleep(time.Second)
	select {
	case <-ch:
	default:
		t.Errorf("The service did not call OnRequest after the Start method was called")
	}
}

func TestStop(t *testing.T) {
	ch := make(chan string)
	srv := newTestService(ch)

	if err := srv.Stop(); err == nil {
		t.Errorf("The service successfully stopped before being started")
	}

	srv.Start()
	if err := srv.Stop(); err != nil || !srv.stopped {
		t.Errorf("The service did not stop successfully after being started")
	}

	select {
	case <-srv.Done():
	default:
		t.Errorf("The service stopped and did not close the Done channel")
	}

	srv.Request(context.TODO(), "testData")
	select {
	case <-ch:
		t.Errorf("The service called OnRequest after the Stop method was called")
	default:
	}
}

func TestLen(t *testing.T) {
	ch := make(chan string)
	srv := newTestService(ch)

	srv.Start()
	defer srv.Stop()

	strs := []string{"str1", "str2", "str3"}
	for _, str := range strs {
		srv.Request(context.TODO(), str)
	}
	time.Sleep(time.Second)

	if l := srv.Len(); l != 2 {
		t.Errorf("Expected 2 requests to be on the queue and Len returned %d", l)
	}

	for i := 0; i < 3; i++ {
		<-ch
	}

	if l := srv.Len(); l != 0 {
		t.Errorf("Expected 0 requests to be on the queue and Len returned %d", l)
	}
}

func TestRequest(t *testing.T) {
	ch := make(chan string)
	srv := newTestService(ch)

	srv.Start()
	defer srv.Stop()
	ctx, cancel := context.WithCancel(context.Background())

	strs := []string{"str1", "str2", "str3"}
	for _, str := range strs {
		srv.Request(ctx, str)
	}

	// Check that the requests are being processed in the correct order
	if str := <-ch; str != strs[0] {
		t.Errorf("Expected %s to be returned and received %s", strs[0], str)
	}

	time.Sleep(time.Second)
	cancel()
	<-ch // Release data from the second request

	select {
	case <-ch:
		t.Errorf("The OnRequest method was called after the context was cancelled")
	default:
	}
}

func TestRateLimit(t *testing.T) {
	ch := make(chan string)
	srv := newTestService(ch)

	srv.SetRateLimit(1)
	srv.Start()
	defer srv.Stop()

	strs := []string{"str1", "str2", "str3"}
	for _, str := range strs {
		srv.Request(context.TODO(), str)
	}

	// The first request is not rate limited
	<-ch
	start := time.Now()
	<-ch
	<-ch
	finish := time.Now()

	if finish.Sub(start) < time.Second {
		t.Errorf("The rate limit was not enforced between calls to the OnRequest method")
	}
}

type testService struct {
	BaseService
	started bool
	stopped bool
	next    chan string
}

func newTestService(next chan string) *testService {
	t := &testService{
		stopped: true,
		next:    next,
	}

	t.BaseService = *NewBaseService(t, "Test")
	return t
}

func (t *testService) OnStart() error {
	t.started = true
	t.stopped = false
	return nil
}

func (t *testService) OnStop() error {
	t.started = false
	t.stopped = true
	return nil
}

func (t *testService) OnRequest(ctx context.Context, args Args) {
	if r, ok := args.(string); ok {
		t.next <- r
		t.CheckRateLimit()
	}
}
