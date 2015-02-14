package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/golangbox/gobox/client/watcher"
	"github.com/golangbox/gobox/structs"
	// "github.com/golangbox/gobox/client/api"
)

// PROBLEMS: No way to tell if a remove event was dir or a file because it can't be os.Stat'ed
//           Can't remove that dir from a watch because Watcher.watches isn't exposed

const (
	dataDirectoryBasename    = ".Gobox"
	serverEndpoint           = "http://requestb.in/1mv9fa41"
	filesystemCheckFrequency = 5
	HASH_ERROR               = 1
)

func startWatcher(dir string) (out chan structs.StateChange, err error) {
	rw, err := watcher.NewRecursiveWatcher(dir)
	if err != nil {
		return out, err

	}
	rw.Run(false)
	fmt.Println("about to fail")
	return rw.Files, err
}

func serverActions() (out chan structs.StateChange, err error) {
	return
}

func fanActionsIn(watcherActions <-chan structs.StateChange,
	serverActions <-chan structs.StateChange) chan structs.StateChange {
	out := make(chan structs.StateChange)
	go func() {
		for {
			select {
			case stateChange := <-watcherActions:
				out <- stateChange
			case stateChange := <-serverActions:
				out <- stateChange
			}
		}
	}()
	return out
}

func writeFileSystemStateToLocalFile(fileSystemState structs.FileSystemState, path string) error {
	jsonBytes, err := json.Marshal(fileSystemState)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, jsonBytes, 0644)
	return err
}

func createGoboxLocalDirectory(path string) {
	if _, err := os.Stat(path); err != nil {
		fmt.Println(err.Error())
		if os.IsNotExist(err) {
			fmt.Println(err)
			fmt.Println("Making directory")
			err := os.Mkdir(path, 0777)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func fetchFileSystemState(path string) (fileSystemState structs.FileSystemState, err error) {
	if _, err := os.Stat(path); err != nil {
		fmt.Println("Making empty data file")
		emptyState := structs.FileSystemState{
			FileActionId: 1,
			State:        make(map[string]structs.File),
		}
		writeFileSystemStateToLocalFile(emptyState, path)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("also here")
	if data != nil {
		err = json.Unmarshal(data, &fileSystemState)
		if err != nil {
			fmt.Println(err)
		}
	}
	return

}

func getSha256FromFilename(filename string) (sha256String string,
	err error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Error reading file for sha256: %s", err)
	}
	h := sha256.New()
	_, err = h.Write(file)
	if err != nil {
		return "", fmt.Errorf("Error writing file to hash for sha256: %s", err)
	}
	byteString := h.Sum(nil)

	sha256String = hex.EncodeToString(byteString)

	return sha256String, nil
}

func fileActionSender(change structs.StateChange) {
	select {
	case <-change.Quit:
		return
	default:

	}

}

func hasher(change structs.StateChange) {
	select {
	case <-change.Quit:
		return
	default:
		h, err := getSha256FromFilename(change.File.Path)
		if err != nil {
			change.Error <- HASH_ERROR
			return
		}
		change.File.Hash = h
		go fileActionSender(change)
	}
	return
}

type ArbitraryFanIn struct {
	Chans  []reflect.SelectCase
	ErrOut <-chan interface{}
}

// func (s *ArbitraryFanIn) AddChannel(chan

func FanNIn(newChannels <-chan chan interface{}, out chan<- interface{}, removeOnEvent bool) {
	var ch <-chan interface{}
	chans := make([]reflect.SelectCase, 0)
	timeout := time.Tick(1000 * time.Millisecond)
	chans = append(chans, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(timeout),
	})
	for {
		select {
		case ch = <-newChannels:
			chans = append(chans, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch),
			})
		default:
			chosen, value, ok := reflect.Select(chans)
			if chosen == 0 {
				fmt.Println("timeout")
				continue
			}
			if removeOnEvent || !ok {
				lastIndex := len(chans) - 1
				chans[chosen] = chans[lastIndex]
				chans = chans[:lastIndex]
			}
			if ok {
				out <- value.Interface()
			}
		}

	}
}

func stephen(dataPath string, stateChanges <-chan structs.StateChange) {
	// spin up a goroutine that will fan in error messages using reflect.select
	// hand it an error channel, and add this to the main select statement
	// do the same thing for done, so I can write a generic fan-n-in function
	fileSystemState, err := fetchFileSystemState(dataPath)
	if err != nil {
		panic("Could not properly retrieve fileSystemState")
	}
	pendingChanges := make(map[string]chan bool)
	newErrors := make((chan (chan interface{})))
	newDones := make((chan (chan interface{})))
	errors := make(chan interface{})
	dones := make(chan interface{})
	go FanNIn(newErrors, errors, true)
	go FanNIn(newDones, dones, true)
	for {
		select {
		case e := <-errors:
			pth := reflect.ValueOf(e).FieldByName("Path")
			fmt.Println("Experienced an error with ", pth)
			// delete(pendingChanges, err

		case change := <-stateChanges:
			fmt.Println(change)
			quitChan := make(chan bool)
			doneChan := make(chan interface{})
			newDones <- doneChan
			errChan := make(chan interface{})
			newErrors <- errChan
			pendingChanges[change.File.Path] = quitChan
			change.Quit = quitChan
			change.Done = doneChan
			change.Error = errChan

		}
	}
	fmt.Println(fileSystemState)

}

func run(path string) {
	goboxDirectory := path
	goboxDataDirectory := fmt.Sprint(goboxDirectory, "/", dataDirectoryBasename)
	createGoboxLocalDirectory(goboxDataDirectory)
	watcherActions, err := startWatcher(path)
	if err != nil {
		panic("Could not start watcher")
	}
	remoteActions, err := serverActions()
	if err != nil {
		panic("Could not properly start remote actions")
	}
	actions := fanActionsIn(watcherActions, remoteActions)

	stephen(goboxDataDirectory+"/data", actions)

	fmt.Println(watcherActions)
	for {
		time.Sleep(1000)
	}

}

func main() {
	run("../sandbox")
}
