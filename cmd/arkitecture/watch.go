package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const pollInterval = 200 * time.Millisecond

type watchEvent int

const (
	eventNone watchEvent = iota
	eventChanged
	eventCreated
	eventDeleted
)

// watchState tracks the input file's presence and modification time so each poll
// tick can be classified. It is kept separate from the loop so the transition
// logic can be tested without timers.
type watchState struct {
	lastMod time.Time
	exists  bool
}

func (w *watchState) next(mod time.Time, present bool) watchEvent {
	switch {
	case present && !w.exists:
		w.exists, w.lastMod = true, mod
		return eventCreated
	case !present && w.exists:
		w.exists = false
		return eventDeleted
	case present && mod.After(w.lastMod):
		w.lastMod = mod
		return eventChanged
	default:
		return eventNone
	}
}

func modTime(path string) (time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

// runWatch processes the input once, then polls it and regenerates on change
// until interrupted. It always returns 0 — watch mode ends only on Ctrl+C.
func runWatch(input, output string, validateOnly, verbose bool, fontSize int, fontFamily string) int {
	fmt.Printf("Watching %s for changes (press Ctrl+C to stop)...\n", input)
	regenerate(input, output, validateOnly, verbose, fontSize, fontFamily)

	mod, present := modTime(input)
	state := watchState{lastMod: mod, exists: present}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			fmt.Println("\nStopping watch mode.")
			return 0
		case <-ticker.C:
			mod, present := modTime(input)
			switch state.next(mod, present) {
			case eventChanged:
				fmt.Printf("[%s] %s changed, regenerating...\n", stamp(), input)
				regenerate(input, output, validateOnly, verbose, fontSize, fontFamily)
			case eventCreated:
				fmt.Printf("[%s] %s created, regenerating...\n", stamp(), input)
				regenerate(input, output, validateOnly, verbose, fontSize, fontFamily)
			case eventDeleted:
				fmt.Printf("[%s] %s deleted (waiting for it to return)...\n", stamp(), input)
			}
		}
	}
}

// regenerate runs one processing pass, reporting status but never exiting, so a
// bad edit doesn't stop the watch.
func regenerate(input, output string, validateOnly, verbose bool, fontSize int, fontFamily string) {
	switch process(input, output, validateOnly, verbose, fontSize, fontFamily) {
	case 0:
		// process() already printed the success line.
	case 1:
		fmt.Fprintf(os.Stderr, "[%s] validation errors (still watching)\n", stamp())
	default:
		fmt.Fprintf(os.Stderr, "[%s] processing failed (still watching)\n", stamp())
	}
}

func stamp() string { return time.Now().Format("15:04:05") }
