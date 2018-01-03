package main

import (
	"sync"
)

type SeenString struct {
	seen map[string]bool
	m    *sync.Mutex
}

func NewSeenString() *SeenString {
	return &SeenString{
		seen: make(map[string]bool),
		m:    &sync.Mutex{},
	}
}

func (s *SeenString) SeenBefore(name string) bool {
	s.m.Lock()
	defer s.m.Unlock()
	if !s.seen[name] {
		s.seen[name] = true
		return false
	} else {
		return true
	}
}

func (s *SeenString) See(name string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.seen[name] = true
}
