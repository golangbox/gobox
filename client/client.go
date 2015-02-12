package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/golangbox/gobox/client/watcher"
	"github.com/golangbox/gobox/structs"
)

// PROBLEMS: No way to tell if a remove event was dir or a file because it can't be os.Stat'ed
//           Can't remove that dir from a watch because Watcher.watches isn't exposed

const (
	dataDirectoryExt         = ".Gobox"
	serverEndpoint           = "http://requestb.in/1mv9fa41"
	filesystemCheckFrequency = 5
	HASH_ERROR               = 1
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
	return rw.Files, err
}

func remoteActions() (out chan structs.StateChange, err error) {
	return
}

func fanActionsIn(watcherActions <-chan structs.StateChange,
	serverActions <-chan structs.StateChange) (out chan structs.StateChange) {
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
	return
}

func writefileSystemStateToLocalFile(fileSystemState map[string]structs.File, path string) error {
	jsonBytes, err := json.Marshal(fileSystemState)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path+"/data", jsonBytes, 0644)
	return err
}

func createGoboxLocalDirectory(path string) {

	_, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Making directory")
		err := os.Mkdir(path, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func fetchFileSystemState(path string) (fileSystemState map[string]structs.File, err error) {
	data, err := ioutil.ReadFile(path + "/data")
	if err != nil {
		return
	}
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
	case q := <-change.Quit:
		return
	default:

	}

}

func hasher(change structs.StateChange) {
	select {
	case q := <-change.Quit:
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

func stephen(dataPath string, stateChanges <-chan structs.StateChange) {
	fileSystemState, err := fetchFileSystemState(dataPath)
	if err != nil {
		panic("Could not properly retrieve fileSystemState")
	}
	pendingChanges := make(map[string]structs.ChannelMessages)

	for {
		select {
		case change := <-stateChanges:
			quitChan := make(chan bool)
			doneChan := make(chan bool)
			errChan := make(chan int)
			messages := structs.ChannelMessages{
				Quit:  quitChan,
				Done:  doneChan,
				Error: errChan,
			}
			pendingChanges[change.File.Path] = messages
			change.Quit = quitChan
			change.Done = doneChan
			change.Error = errChan
			if change.IsCreate {

			}
		}
	}

}

func run(path string) {
	goboxDirectory := path
	goboxDataDirectory := goboxDirectory + dataDirectoryExt

	watcherActions, err := startWatcher(path)

	createGoboxLocalDirectory(goboxDataDirectory)

	if err != nil {
		panic("Could not start watcher")
	}
	remoteActions, err := startWatcher(path)
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
