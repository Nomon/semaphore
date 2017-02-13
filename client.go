package semaphore

import "context"

// Client is an interface to distributed locking on a semaphore state.
type Client interface {
	// Load loads the semaphore state from remote storage
	Load(context.Context) (*State, error)
	// Save attempts to save the state to remote storge, in case of a conflict
	// ErrRemoteConflict is returned.
	Save(context.Context, State) error
	// Wait blocks until the state at remote is newer than the passed in state,
	// used to wait for a slot in the semaphore.
	Wait(context.Context, State) error
}
