package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golangbox/gobox/client/structs"
	"github.com/golangbox/gobox/client/watcher"
)

// PROBLEMS: No way to tell if a remove event was dir or a file because it can't be os.Stat'ed
//           Can't remove that dir from a watch because Watcher.watches isn't exposed

const (
	goBoxDirectory           = "."
	goBoxDataDirectory       = ".GoBox"
	serverEndpoint           = "http://requestb.in/1mv9fa41"
	filesystemCheckFrequency = 5
)

func startWatcher(dir string) (out chan structs.StateChange, err error) {
	if err != nil {
		log.Fatal(err)
	}
	rw, err := watcher.NewRecursiveWatcher(dir)
	if err != nil {
		log.Println(err.Error())
		log.Fatal("Couldn't start a recursive watcher")

	}
	rw.Run(false)
	go func() {
		for {
			fileEv := <-rw.Files
			fmt.Println(fileEv)

		}
	}()
	go func() {
		for {
			foldEv := <-rw.Folders
			fmt.Println(foldEv)
		}
	}()

	for {
		time.Sleep(1000)
	}
	return rw.Files, err
}

func updateRequest() (out chan structs.StateChange) {
	return
}

func fanActionsIn(watcherChanges <-chan structs.StateChange,
	serverChanges <-chan structs.StateChange) (out chan structs.StateChange) {
	go func() {
		for {
			select {
			case stateChange := <-watcherChanges:
				out <- stateChange
			case stateChange := <-serverChanges:
				out <- stateChange
			}
		}
	}()
	return
}

func stephen(stateChanges <-chan structs.StateChange) {
	//
}

func run(path string) {
	watcherActions, _ := startWatcher(path)
	fmt.Println(watcherActions)
	for {
		time.Sleep(1000)
	}

}

func main() {
	run("../test")
}
