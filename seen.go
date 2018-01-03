package main

import (
	"sync"
)

// SeenString is a synchronized set of strings. Once See is called
// with a string, SeenBefore will report that it has been seen.
type SeenString struct {
	seen map[string]bool
	m    *sync.Mutex
}

// NewSeenString creates a new SeenString
func NewSeenString() *SeenString {
	return &SeenString{
		seen: make(map[string]bool),
		m:    &sync.Mutex{},
	}
}

// SeenBefore reports whether given string was seen before.
func (s *SeenString) SeenBefore(name string) bool {
	s.m.Lock()
	defer s.m.Unlock()
	if !s.seen[name] {
		s.seen[name] = true
		return false
	}
	return true
}

// See marks the string as seen.
func (s *SeenString) See(name string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.seen[name] = true
}
