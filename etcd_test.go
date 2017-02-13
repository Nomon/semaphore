package semaphore

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/etcd/client"
)

func testEtcdClient(t *testing.T) Client {
	opt := client.Config{
		Transport: client.DefaultTransport,
	}
	if endpoints := os.Getenv("ETCD"); endpoints != "" {
		opt.Endpoints = strings.Split(endpoints, ",")
	} else {
		t.Skip("No etcd configured")
	}
	c, err := client.New(opt)
	if err != nil {
		t.Fatal(err)
	}
	ec, err := NewEtcdClient(c, fmt.Sprintf("/semaphore_tests_%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	return ec
}

func lockUnlock(t *testing.T, s *Semaphore, fn func(string) error) {
	if err := s.Acquire(context.Background()); err != nil {
		t.Error(err)
	}
	if err := fn(s.holder); err != nil {
		t.Error(err)
	}
	if err := s.Release(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestEtcdSemaphore(t *testing.T) {
	ec := testEtcdClient(t)
	sems := []*Semaphore{
		NewSemaphore(ec, "holder1"),
		NewSemaphore(ec, "holder2"),
		NewSemaphore(ec, "holder3"),
	}

	var wg sync.WaitGroup
	var calls []string
	wg.Add(len(sems))
	for _, sem := range sems {
		go lockUnlock(t, sem, func(h string) error {
			calls = append(calls, h)
			wg.Done()
			return nil
		})
	}
	expected := []string{"holder1", "holder2", "holder3"}
	wg.Wait()
	for _, e := range expected {
		if !stringInSlice(e, calls) {
			t.Fatalf("Holder %s never aquired lock", e)
		}
	}
}

func TestEtcdMaxHolders(t *testing.T) {
	ec := testEtcdClient(t)
	sem := NewSemaphore(ec, "sem1")
	limit := 10
	sem.SetLimit(context.Background(), limit)
	var wg sync.WaitGroup
	maxHolders := int32(0)

	for i := 0; i < limit*10; i++ {
		wg.Add(1)
		ns := NewSemaphore(ec, fmt.Sprintf("sem_holder%d", i))
		go func(sem *Semaphore) {
			defer wg.Done()
			if err := sem.Acquire(context.Background()); err != nil {
				t.Error(err)
			}
			atomic.AddInt32(&maxHolders, 1)
			time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
			cur := atomic.LoadInt32(&maxHolders)
			if cur > int32(limit) {
				t.Error("too many holders", cur)
			}
			atomic.AddInt32(&maxHolders, -1)
			if err := sem.Release(context.Background()); err != nil {
				t.Error(err)
			}
		}(ns)
	}
	wg.Wait()
}

func TestEtcdSemaphoreContext(t *testing.T) {
	ec := testEtcdClient(t)
	sem := NewSemaphore(ec, "h1")
	sem2 := NewSemaphore(ec, "h2")

	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		ctx, cf := context.WithDeadline(context.Background(), time.Now().Add(300*time.Millisecond))
		defer cf()
		if err := sem2.Acquire(ctx); err == nil {
			t.Error("Expected context deadline error to propagate to semaphore")
		}
		wg.Done()
	}()
	wg.Wait()
}
