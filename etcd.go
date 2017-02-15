package semaphore

import (
	"context"
	"encoding/json"
	"errors"
	"path"

	"github.com/coreos/etcd/client"
)

// DefaultPrefix is the prefix used for semaphores if no prefix is provided
var DefaultPrefix = "semaphores"

// DefaultKey is the default key to use for the semaphore document
var DefaultKey = "semaphore"

// ErrRemoteConflict is returned by Save() when remote state index is greater than local state index.
var ErrRemoteConflict = errors.New("Failed to save remote state, conflict.")

// EtcdClient is a client used to distribute the semaphore.
type EtcdClient struct {
	client  client.KeysAPI
	keypath string
}

// NewEtcdClient creates a new EtcdClient. The path parameter defines
// the etcd key in which the client will manipulate the semaphore. If the
// key is the empty, the default path of /semaphores will be used.
func NewEtcdClient(c client.Client, key string) (*EtcdClient, error) {
	k := path.Join(DefaultPrefix, DefaultKey)
	if key != "" {
		k = key
	}
	return NewEtcdKeysAPI(client.NewKeysAPI(c), k)
}

func NewEtcdKeysAPI(c client.KeysAPI, key string) (*EtcdClient, error) {
	k := path.Join(DefaultPrefix, DefaultKey)
	if key != "" {
		k = key
	}
	elc := &EtcdClient{c, k}
	if err := elc.init(); err != nil {
		return nil, err
	}
	return elc, nil
}

// Reset force resets the semaphore to default empty state
func (c *EtcdClient) Reset() error {
	s := NewState()
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if _, err := c.client.Set(context.Background(), c.keypath, string(b), nil); err != nil {
		return err
	}
	return nil
}

// create ensures that a valid state exists in the remote at c.keypath
func (c *EtcdClient) init() error {
	s := NewState()

	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if _, err := c.client.Create(context.Background(), c.keypath, string(b)); err != nil {
		eerr, ok := err.(client.Error)
		if ok && eerr.Code == client.ErrorCodeNodeExist {
			return nil
		}
		return err
	}

	return nil
}

// Load fetches the Semaphore state from etcd
func (c *EtcdClient) Load(ctx context.Context) (*State, error) {
	resp, err := c.client.Get(ctx, c.keypath, nil)
	if err != nil {
		return nil, err
	}
	s := NewState()
	if err = json.Unmarshal([]byte(resp.Node.Value), s); err != nil {
		return nil, err
	}
	s.Index = resp.Node.ModifiedIndex

	return s, nil
}

// Save attempts to save the local state into the remote state using current index. If the remote
// state is newer than local state, returns ErrRemoteConflict
func (c *EtcdClient) Save(ctx context.Context, s State) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	// use PrevIndex to ensure local state matches remote state before changing it.
	setopts := &client.SetOptions{
		PrevIndex: s.Index,
	}

	_, err = c.client.Set(ctx, c.keypath, string(b), setopts)
	// did it fail due to the remote state being newer than local state?
	if isCompareFailed(err) {
		return ErrRemoteConflict
	}
	return err
}

// Wait waits for remote state index changes
func (c *EtcdClient) Wait(ctx context.Context, s State) error {
	watcher := c.client.Watcher(c.keypath, &client.WatcherOptions{
		AfterIndex: s.Index,
	})
	// block untill the state is/changes to newer than s.Index
	_, err := watcher.Next(ctx)
	return err
}

// isCompareFailed returns true if the ETCD request failed to index conflict,
// meaning that the remote state is newer than local state.
func isCompareFailed(err error) bool {
	if eerr, ok := err.(client.Error); ok {
		if eerr.Code == client.ErrorCodeTestFailed {
			return true
		}
	}
	return false
}
