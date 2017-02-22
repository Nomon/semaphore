package semaphore

import (
	"errors"
	"sort"
)

// State is the semaphores serialized state that is passed to client for syncing.
type State struct {
	// Index is an monotonically increasing number that increases any time it is changed.
	// it is used to make sure operations applied against remote state are not overwritten
	// by changes against an older state.
	Index     uint64   `json:"-"`
	// Semaphore is amount of locks left to give out in the state
	Semaphore int      `json:"semaphore"`
	// Limit limits the max number of concurrent locks
	Limit     int      `json:"limit"`
	// Holders contain all the current semaphore lock holders
	Holders   []string `json:"holders"`
}

// NewState returns a semaphore state in its default state, unlocked with max 1 lock holders.
func NewState() *State {
	return &State{
		Holders:   make([]string, 0),
		Limit:     1,
		Semaphore: 1,
	}
}

func  (s *State) Copy() *State {
	cp := *s
	cp.Holders = make([]string, len(s.Holders))
	if len(s.Holders) > 0 {
		copy(cp.Holders, s.Holders)
	}
	return cp
}
// setLimit sets the states limit to the passed in limit and adjusts the semaphore count accordingly.
func (s *State) setLimit(limit int) {
	change := s.Limit - limit
	s.Semaphore = s.Semaphore - change
	s.Limit = limit
}

// addHolder adds the holder to state, returning true if the state was changed.
func (s *State) addHolder(holder string) bool {
	pos := sort.SearchStrings(s.Holders, holder)
	if len(s.Holders) == pos {
		s.Holders = append(s.Holders, holder)
		return true
	}
	if s.Holders[pos] == holder {
		return false
	}
	s.Holders = append(s.Holders[:pos], append([]string{holder}, s.Holders[pos:]...)...)
	return true
}

// removeHolder removes a holder from state, returns true if the state was changed.
func (s *State) removeHolder(holder string) bool {
	pos := sort.SearchStrings(s.Holders, holder)
	if pos < len(s.Holders) && s.Holders[pos] == holder {
		s.Holders = append(s.Holders[:pos], s.Holders[pos+1:]...)
		return true
	}
	return false
}

// lock aquires a slot in the semaphore state for holder
func (s *State) lock(holder string) error {
	// holder cannot lock state, semaphore is all empty
	// error signals for Semaphore to wait for state to change before retrying.
	if s.Semaphore <= 0 {
		return errors.New("Semaphore locked")
	}
	// if holder was 
	if s.addHolder(holder) {
		s.Semaphore = s.Semaphore - 1
	}
	return nil
}

// unlock unlocks the semaphore for the owner
func (s *State) unlock(holder string) error {
	// if the holder was actually remove, increase semaphore
	if s.removeHolder(holder) {
		s.Semaphore = s.Semaphore + 1
	}
	return nil
}
