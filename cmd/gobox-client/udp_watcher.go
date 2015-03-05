package main

import (
	"bufio"
	"io"
)

// Interface used by the UDP Listener
type RemoteEventCollector interface {
	NewRemoteEvent(stateChange)
}

// Called by Server Notifications
func (s collector) NewRemoteEvent(c stateChange) {
	s.collect <- c
}

// TODO move to tests
var _ RemoteEventCollector = collector{}

type readerWatcher struct {
	// TODO There should probly be some communication if
	//      watcher encounters an error while it's running

	// Stores the error that caused watcher to exit
	err error
}

func (w *readerWatcher) watchFrom(r io.Reader, sink RemoteEventCollector) {
	go func() {
		scanner := bufio.NewScanner(r)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			sink.NewRemoteEvent(stateChange(scanner.Text()))
		}

		w.err = scanner.Err()
	}()
}
