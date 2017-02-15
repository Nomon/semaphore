package semaphore

import (
	"context"
	"os"
)

// Semaphore is used to do semaphore locking, allowing N configured concurrent locks.
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
func (s *Semaphore) Acquire(ctx context.Context) error {
	// apply the lock to the semaphore
	err := s.applyState(ctx, func() error {
		return s.state.lock(s.holder)
	})
	return err
}

// Release releases the aquired slot
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
	local := s.state
	// load the current remote state from the client
	remote, err := s.client.Load(ctx)
	if err != nil {
		return err
	}

	// apply remote state to local state
	local.apply(remote)

	// apply desired changes to the local state
	if err := f(); err != nil {
		// the operation fails to apply against current state, block until the stage changes to a more
		// favorable one.
		if err = s.client.Wait(ctx, *local); err != nil {
			return err
		}
		// state has changed, retry apply
		return s.applyState(ctx, f)
	}

	// The state change was applied to local state, save local state to remote
	if err = s.client.Save(ctx, *local); err != nil {
		// if the remote state has changed and no longer match our local state, refresh local state
		// and retry operation
		if err == ErrRemoteConflict {
			return s.applyState(ctx, f)
		}
		return err
	}
	return nil
}

// stringInSlice checks if str is part of slice
func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
