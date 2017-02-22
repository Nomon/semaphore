package semaphore

import (
	"context"
	"os"
	"strings"
)

// Semaphore is used to do semaphore locking, allowing N configured concurrent locks,
// adjustable with SetLimit. 
type Semaphore struct {
	client Client
	holder string
	index  uint64
	state  *State
}

// NewSemaphore returns a new Semaphore backed by Client. Holder is the
// holder id of this semaphore instance, defaults to os.Hostname().
func NewSemaphore(c Client, holder string) *Semaphore {
	if holder == "" {
		holder, _ = os.Hostname()
	}
	s := &Semaphore{
		holder: holder,
		client: c,
		state:  NewState(),
	}
	return s
}

// Reset resets the semaphore releasing all locks.
func (s *Semaphore) Reset() error {
	return s.client.Reset()
}

// SetLimit alters the concurrent lock holders limit for the semaphore
func (s *Semaphore) SetLimit(ctx context.Context, limit int) error {
	err := s.applyState(ctx, func() error {
		s.state.setLimit(limit)
		return nil
	})
	return err
}

// Acquire aquires a lock in the semaphore, if the limit is reached it blocks waiting for a slot.
// Loads the latest state with Client and tries to apply a lock operations against it. If the operation is possible
// ie. the semaphore having less than Limit holders, a save of the local modified state is attempted.
// if the save fails for remote state having already changed, the new state is loaded and the operation is retried.
// Canceling the passed in context.Context will unblock the operation and the context  
func (s *Semaphore) Acquire(ctx context.Context) error {
	// apply the lock to the semaphore
	err := s.applyState(ctx, func() error {
		return s.state.lock(s.holder)
	})
	return err
}

// Release does pretty much the opposite of Acquire but it usually is a fast operation, slowed down 
// only by retries due to state conflicts when attempting to save the state with Client. 
func (s *Semaphore) Release(ctx context.Context) error {
	err := s.applyState(ctx, func() error {
		return s.state.unlock(s.holder)
	})
	return err
}

// applyState loads the semaphore remote state with client, calls user supplied function
// and attempts to apply the state if no error is returned from the supplied function.
// if user supplied function returns an error, applyState will wait for remote state to
// change, once changed the state apply is retried against the new state.
func (s *Semaphore) applyState(ctx context.Context, f func() error) (err error) {
	// load the current remote state from the client
	remote, err := s.client.Load(ctx)
	if err != nil {
		return err
	}
	s.state = remote
	
	// apply changes (lock/unlock) to the local state. 
	if err := f(); err != nil {
		// the operation failed to apply against current local state we just loaded,
		// block until remote.Index > local.Index and retry
		if err = s.client.Wait(ctx, *s.state); err != nil {
			return err
		}
		// state has changed, lets retry load + apply changes
		return s.applyState(ctx, f)
	}

	// The fn is complete with its state operations, save local state to remote with client.
	if err = s.client.Save(ctx, *s.state); err != nil {
		// if the remote state has changed, someone else has locked or limit has been adjusted etc
		// and local state is out of sync. resync local with remote and retry changes+save
		if err == ErrRemoteConflict {
			return s.applyState(ctx, f)
		}
		return err
	}
	return nil
}

