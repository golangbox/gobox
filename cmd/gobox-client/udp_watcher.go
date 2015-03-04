package main

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
