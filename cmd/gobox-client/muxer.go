package main

import "log"

type stateChange interface{}

type collectHaltRequest struct {
	hasHalted <-chan struct{}
}

type collector struct {
	// Changes sent to this channel are collected
	collect chan<- stateChange

	// TODO needs to stop
}

// Starts a go routine that will collect state changes
// and send them out the channel returned.
func (s *collector) startCollecting() (changes <-chan stateChange) {
	collectCh := make(chan stateChange)
	s.collect = collectCh
	var newChange <-chan stateChange = collectCh

	changesOutCh := make(chan stateChange)
	changes = changesOutCh
	var fifo chan<- stateChange

	go func() {
		// buffer to collect changes in
		var changes []stateChange

		// first change to be sent out
		var change stateChange

		for {
			if change == nil && len(changes) > 0 {
				change = changes[0]
				changes = changes[1:]
			}

			if change == nil {
				select {
				case c := <-newChange:
					changes = append(changes, c)
				}

			} else {
				select {
				case c := <-newChange:
					changes = append(changes, c)

				case fifo <- change:
					change = nil
				}
			}
		}
	}()

	return changes
}

type changeId int

type executionError struct {
	change executableChange
	err    error
}

type executableChange interface {
	Id() changeId
	Exec(errCh chan<- executionError, doneCh chan<- executableChange)
	Halt()
}

type changeHaltRequest struct {
	hasHalted <-chan executableChange
}

type upload struct {
	id changeId
	stateChange

	requestHalt <-chan changeHaltRequest
}

func (c upload) Id() changeId { return c.id }
func (c upload) Exec(errCh chan<- executionError, doneCh chan<- executableChange) {
	go func() {
		// TODO Do upload
		doneCh <- c
	}()
}
func (upload) Halt() {
}

type download struct {
	id changeId
	stateChange

	requestHalt <-chan changeHaltRequest
}

func (c download) Id() changeId { return c.id }
func (c download) Exec(errCh chan<- executionError, doneCh chan<- executableChange) {
	go func() {
		// TODO Do download
		doneCh <- c
	}()
}
func (download) Halt() {
}

func newExecutableChange(id changeId, sc stateChange) executableChange {
	// TODO Process change, sc
	//      - Is it an upload?
	//      - Is it a download?

	// TODO return the correct type of exec change
	return upload{id, sc, nil}
}

type executorHaltRequest struct {
	hasHalted chan<- struct{}
}

type executor struct {
	// Used by the Halt() method to stop the executor
	requestHalt chan<- executorHaltRequest
}

func (e *executor) executeFrom(changesIn <-chan stateChange) {
	haltCh := make(chan executorHaltRequest)
	e.requestHalt = haltCh
	var haltRequested <-chan executorHaltRequest = haltCh

	go func() {
		errCh := make(chan executionError)
		var executionErr <-chan executionError = errCh

		doneCh := make(chan executableChange)
		var executionComplete <-chan executableChange = doneCh

		nextId := func() func() changeId {
			var nextId changeId = 0
			return func() changeId {
				defer func() { nextId++ }()
				id := nextId
				return id
			}
		}()

		runningChanges := make(map[changeId]executableChange)

		for {
			select {
			case c := <-changesIn:
				ec := newExecutableChange(nextId(), c)
				// TODO Should I cancel any running changes because of ec?

				// Non blocking
				ec.Exec(errCh, doneCh)

				runningChanges[ec.Id()] = ec

			case e := <-executionErr:
				log.Println(e.err)

			case c := <-executionComplete:
				delete(runningChanges, c.Id())

			case r := <-haltRequested:
				// TODO Cleanup
				r.hasHalted <- struct{}{}
			}
		}
	}()
}

func (e executor) halt() {
	hasHalted := make(chan struct{})
	e.requestHalt <- executorHaltRequest{hasHalted}
	<-hasHalted
}
