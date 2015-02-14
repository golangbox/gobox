package main

import (
	"fmt"
	"github.com/golangbox/gobox/boxtools"
	"reflect"
	"testing"
)

func TestStartWatcher(t *testing.T) {
	_, err := startWatcher("/billybob")
	if err == nil {
		t.Log("startWatcher must fail on invalid directory")
		t.FailNow()
	}

	ch, err := startWatcher("../sandbox")
	if err != nil {
		t.Log("startWatcher didn't work on a valid directory")
		t.FailNow()
	}

	if reflect.ValueOf(ch).Kind() != reflect.Chan {
		t.Log("Return value from startWatcher must be a channel")
		t.FailNow()
	}
	go boxtools.SimulateFilesystemChanges("../sandbox", 10, 5, 3)
	for i := 0; i < 18; i++ {
		fmt.Println(<-ch)
	}
}

func TestServerActions(t *testing.T) {

}

func TestFanActionsIn(t *testing.T) {

}

func TestStephen(t *testing.T) {

}
