package semaphore

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

type testClient struct {
	m   sync.Mutex
	s   *State
	c   *sync.Cond
	err error
}

func newTestClient(err error, s *State) Client {
	tc := &testClient{
		s:   s,
		err: err,
	}
	tc.c = sync.NewCond(&tc.m)
	return tc
}

func (tc *testClient) SetLimit(ctx context.Context, l int) error {
	tc.m.Lock()
	defer tc.m.Unlock()
	tc.s.setLimit(l)
	return nil
}

func (tc *testClient) Load(ctx context.Context) (*State, error) {
	tc.m.Lock()
	defer tc.m.Unlock()
	holders := tc.s.Holders
	cp := *tc.s
	cp.Holders = make([]string, 0)
	for _, h := range holders {
		cp.Holders = append(cp.Holders, h)
	}

	return &cp, tc.err
}

func (tc *testClient) Save(ctx context.Context, s State) error {
	tc.m.Lock()
	defer tc.m.Unlock()

	if tc.s.Index > s.Index {
		return ErrRemoteConflict
	}
	s.Index++
	tc.s = &s
	tc.c.Broadcast()
	return nil
}

func (tc *testClient) Wait(ctx context.Context, s State) error {
	tc.m.Lock()
	if tc.s.Index >= s.Index {
		tc.m.Unlock()
		return nil
	}
	tc.c.Wait()
	tc.m.Unlock()
	return tc.Wait(ctx, s)
}

func TestSemaphore(t *testing.T) {
	c := newTestClient(nil, NewState())
	sem := NewSemaphore(c, "test1")
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sem.holder != "test1" {
		t.Fatal("invalid holder")
	}
	if len(sem.state.Holders) != 1 {
		t.Fatal("Invalid jolder count", len(sem.state.Holders))
	}
	if err := sem.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sem.state.Holders) != 0 {
		t.Fatal("Invalid jolder count", len(sem.state.Holders))
	}
}

func TestSemaphoreN(t *testing.T) {
	c := newTestClient(nil, NewState())
	sem := NewSemaphore(c, "test1")
	var wg sync.WaitGroup
	if err := sem.SetLimit(context.Background(), 10); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := NewSemaphore(c, fmt.Sprintf("holder_%d", i))
			if err := s.Acquire(context.Background()); err != nil {
				t.Error(err)
			}
			if err := s.Release(context.Background()); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	state, _ := c.Load(context.Background())
	if len(state.Holders) > 0 {
		t.Fatal("Invalid state tracking, unexpected holders", state.Holders)
	}
}
