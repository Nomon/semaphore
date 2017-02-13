package semaphore

import (
	"errors"
	"sort"
)

// State is the semaphores serialized state that is passed to client for syncing.
type State struct {
	Index     uint64   `json:"-"`
	Semaphore int      `json:"semaphore"`
	Limit     int      `json:"limit"`
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

// hasHolder returns true if the state already contains holder
func (s *State) hasHolder(holder string) bool {
	for _, h := range s.Holders {
		if h == holder {
			return true
		}
	}
	return false
}

// lock aquires a slot in the semaphore state for holder
func (s *State) lock(holder string) error {
	if s.hasHolder(holder) {
		// already locked to us.
		return nil
	}
	if s.Semaphore <= 0 {
		return errors.New("Semaphore locked")
	}
	if s.addHolder(holder) {
		s.Semaphore = s.Semaphore - 1
	}
	return nil
}

// unlock unlocks the semaphore for the owner
func (s *State) unlock(holder string) error {
	if s.removeHolder(holder) {
		s.Semaphore = s.Semaphore + 1
	}
	return nil
}

func (s *State) apply(state *State) {
	s.Index = state.Index
	s.Semaphore = state.Semaphore
	s.Limit = state.Limit
	if state.Holders == nil {
		s.Holders = make([]string, 0)
	} else {
		s.Holders = state.Holders
	}
}
