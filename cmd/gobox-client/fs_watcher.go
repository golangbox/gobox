package main

// Interface used by the Fs Watcher
type FsEventCollector interface {
	NewFsEvent(stateChange)
}

// Called by Watcher
func (s collector) NewFsEvent(c stateChange) {
	s.collect <- c
}

// TODO move to tests
var _ FsEventCollector = collector{}
