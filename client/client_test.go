package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/structs"

	"path/filepath"
	"reflect"
	"testing"
)

const (
	sandboxDir = "../clientSandbox"
)

func TestStartWatcher(t *testing.T) {
	ignores := make(map[string]bool)
	ignores[".Gobox"] = true
	abspath, err := filepath.Abs(sandboxDir)
	if err != nil {
		t.Log("Could not clean up properly")
		t.FailNow()
	}
	err = boxtools.CleanTestFolder(abspath, ignores, true)
	if err != nil {
		t.Log("foo")
		t.Log(err.Error())
		t.FailNow()
	}

	initScanDone := make(chan struct{})
	_, err = startWatcher("/baddir", initScanDone)
	if err == nil {
		t.Log("startWatcher must fail on invalid directory")
		t.FailNow()
	}

	watcherActions, err := startWatcher(sandboxDir, initScanDone)
	if err != nil {
		t.Log("startWatcher didn't work on a valid directory")
		t.FailNow()
	}

	if reflect.ValueOf(watcherActions).Kind() != reflect.Chan {
		t.Log("Return value from startWatcher must be a channel")
		t.FailNow()
	}
	select {
	case <-watcherActions:
		t.Log("Should not be able to read a value before initScanDone is signaled")
		t.FailNow()
	default:
		break
	}

	initScanDone <- struct{}{}

	boxtools.SimulateFilesystemChanges(sandboxDir, 3, 3, 3)
	time.Sleep(1 * time.Second)
	fmt.Println("After sleep")

	// check creates
	for i := 0; i < 3; i++ {
		select {
		case action := <-watcherActions:
			if !action.IsCreate {
				t.Log("IsCreate is wrong")
				t.FailNow()
			} else if !action.IsLocal {
				t.Log("IsLocal is wrong")
				t.FailNow()
			}
		default:
			t.Log("Ran out of values to read")
			t.FailNow()
		}

	}

	// check modifies
	for i := 0; i < 3; i++ {
		select {
		case action := <-watcherActions:
			if !action.IsCreate {
				t.Log("IsCreate is wrong")
				t.FailNow()
			} else if !action.IsLocal {
				t.Log("IsLocal is wrong")
				t.FailNow()
			}
		default:
			t.Log("Ran out of values to read")
			t.FailNow()
		}

	}

	// check deletes
	for i := 0; i < 3; i++ {
		select {
		case action := <-watcherActions:
			// checks for delete
			if action.IsCreate {
				t.Log("IsCreate is wrong")
				t.FailNow()
			} else if !action.IsLocal {
				t.Log("IsLocal is wrong")
				t.FailNow()
			}
		default:
			t.Log("Ran out of values to read")
			t.FailNow()
		}

	}

	select {
	case <-watcherActions:
		t.Log("Should not be able to get another value off of here")
		t.FailNow()
	default:
		break
	}

	return
}

func TestServerActions(t *testing.T) {

}

func TestFanActionsIn(t *testing.T) {
	ch1, ch2, ch3 := make(chan structs.StateChange),
		make(chan structs.StateChange),
		make(chan structs.StateChange)

	out := fanActionsIn(ch1, ch2, ch3)
	numRead := 0
	go func() { ch1 <- structs.StateChange{} }()
	go func() { ch2 <- structs.StateChange{} }()
	go func() { ch3 <- structs.StateChange{} }()
	timeout := time.Tick(1000 * time.Millisecond)
	timedOut := false
	for !timedOut {
		select {
		case <-out:
			numRead++
		case <-timeout:
			timedOut = true
		}
	}
	if numRead != 3 {
		t.Log("Wrong number of messages from out channel")
		t.FailNow()
	}

}

func TestArbitraryFanIn(t *testing.T) {
	validNums := make(map[int]bool)
	newChans := make(chan chan interface{})
	out := make(chan interface{})
	arbitraryFanIn(newChans, out, true)
	chans := make([]chan interface{}, 0)
	for i := 0; i < 10; i++ {
		chans = append(chans, make(chan interface{}, 1))
		newChans <- chans[i]
	}

	for i := 0; i < 5; i++ {
		chans[i] <- interface{}(i)
		validNums[i] = true
	}

	for i := 10; i < 20; i++ {
		chans = append(chans, make(chan interface{}, 1))
		newChans <- chans[i]
		chans[i] <- interface{}(i)
		validNums[i] = true
	}

	timeout := time.Tick(1000 * time.Microsecond)
	timedOut := false
	for !timedOut {
		select {
		case v := <-out:
			if _, found := validNums[v.(int)]; !found {
				t.Log("Message not in list of valid numbers")
				t.FailNow()
			}
		case <-timeout:
			timedOut = true
		}
	}
}

func TestWriteError(t *testing.T) {
	// with proper params, a correct message shows up on the recieving side of channel
	errorChannel := make(chan interface{})
	doneChannel := make(chan interface{})
	err := errors.New("Oh Noez!")
	change := structs.StateChange{
		File:  structs.File{},
		Error: errorChannel,
		Done:  doneChannel,
	}
	go writeError(err, change, "TestWriteError")
	// wait for a minute to be sure the message is waiting.
	time.Sleep(100 * time.Millisecond)
	select {
	case msg := <-errorChannel:
		if reflect.TypeOf(msg) != reflect.TypeOf(
			interface{}(structs.ErrorMessage{})) {
			t.Log("Message should be structs.ErrorMessage")
			t.FailNow()
		}
		break
	default:
		t.Log("Should have a message to read")
		t.FailNow()
	}
}

func TestStephen(t *testing.T) {

}
