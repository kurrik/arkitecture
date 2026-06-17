package main

import (
	"testing"
	"time"
)

func TestWatchStateTransitions(t *testing.T) {
	base := time.Unix(1_000_000, 0)
	s := watchState{lastMod: base, exists: true}

	if ev := s.next(base, true); ev != eventNone {
		t.Errorf("unchanged file -> %v, want eventNone", ev)
	}

	later := base.Add(time.Second)
	if ev := s.next(later, true); ev != eventChanged {
		t.Errorf("newer modtime -> %v, want eventChanged", ev)
	}

	if ev := s.next(later, false); ev != eventDeleted {
		t.Errorf("file removed -> %v, want eventDeleted", ev)
	}
	if ev := s.next(later, false); ev != eventNone {
		t.Errorf("still absent -> %v, want eventNone", ev)
	}

	newer := later.Add(time.Second)
	if ev := s.next(newer, true); ev != eventCreated {
		t.Errorf("file returns -> %v, want eventCreated", ev)
	}
	if ev := s.next(newer, true); ev != eventNone {
		t.Errorf("unchanged after recreate -> %v, want eventNone", ev)
	}
}
