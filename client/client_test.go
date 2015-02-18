package main

import (
	"fmt"
	"time"

	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/structs"

	"path/filepath"
	"reflect"
	"testing"
)

const (
	sandboxDir = "../sandbox"
)

func TestStartWatcher(t *testing.T) {
	_, err := startWatcher("/billybob")
	if err == nil {
		t.Log("startWatcher must fail on invalid directory")
		t.FailNow()
	}

	ch, err := startWatcher(sandboxDir)
	if err != nil {
		t.Log("startWatcher didn't work on a valid directory")
		t.FailNow()
	}

	if reflect.ValueOf(ch).Kind() != reflect.Chan {
		t.Log("Return value from startWatcher must be a channel")
		t.FailNow()
	}
	go boxtools.SimulateFilesystemChanges(sandboxDir, 10, 5, 0)
	for i := 0; i < 18; i++ {
		fmt.Println(<-ch)
		fmt.Println(i)

	}
	ignores := make(map[string]bool)
	ignores[".Gobox"] = true
	abspath, err := filepath.Abs(sandboxDir)
	if err != nil {
		t.Log("Could not clean up properly")
		t.FailNow()
	}
	err = boxtools.CleanTestFolder(abspath, ignores, true)
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	return
}

func TestServerActions(t *testing.T) {

}

func TestFanActionsIn(t *testing.T) {

	ch1, ch2 := make(chan structs.StateChange), make(chan structs.StateChange)
	out := fanActionsIn(ch1, ch2)
	numRead := 0
	go func() { ch1 <- structs.StateChange{} }()
	go func() { ch2 <- structs.StateChange{} }()
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
	if numRead != 2 {
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
			fmt.Println(v, v.(int))
			if _, found := validNums[v.(int)]; !found {
				fmt.Println(found)
				t.FailNow()
			}
		case <-timeout:
			timedOut = true
		}
	}
}

func TestStephen(t *testing.T) {
	run(sandboxDir)
	go boxtools.SimulateFilesystemChanges(sandboxDir, 10, 5, 0)
	for {
		time.Sleep(1000)
	}
}

func TestHasherQuitsProperly(t *testing.T) {

}
