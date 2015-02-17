package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golangbox/gobox/client/api"
	"github.com/golangbox/gobox/client/watcher"
	"github.com/golangbox/gobox/structs"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"time"
	// "github.com/golangbox/gobox/client/api"
)

// TODO
// get client/server sessions set up
// sync a damn file
// work on download branch

// PROBLEMS: No way to tell if a remove event was dir or a file because it can't be os.Stat'ed
//           Can't remove that dir from a watch because Watcher.watches isn't exposed

var client api.Api

const (
	dataDirectoryBasename    = ".Gobox"
	serverEndpoint           = "http://requestb.in/1mv9fa41"
	filesystemCheckFrequency = 5
	HASH_ERROR               = 1
)

func writeError(err error, change structs.StateChange, function string) {
	change.Error <- structs.ErrorMessage{
		Error:    err,
		File:     change.File,
		Function: function,
	}
	close(change.Done)
}

// it may be easier to just have this function make a FileAction, but ignore for now
func writeDone(change structs.StateChange, fa structs.FileAction) {
	change.Done <- fa
	close(change.Error)
}

func startWatcher(dir string) (out chan structs.StateChange, err error) {
	rw, err := watcher.NewRecursiveWatcher(dir)
	if err != nil {
		return out, err

	}
	rw.Run(false)
	return rw.Files, err
}

// type FileAction struct {
// 	Id           int64
// 	ClientId     int64
// 	IsCreate     bool
// 	CreatedAt    time.Time
// 	PreviousHash string
// 	File         File
// 	FileId       int64
// }

// type StateChange struct {
// 	File         File
// 	IsCreate     bool
// 	IsLocal      bool
// 	Quit         <-chan bool
// 	Done         chan<- interface{}
// 	Error        chan<- interface{}
// 	PreviousHash string
// }

func createServerStateChange(fa structs.FileAction) (change structs.StateChange) {
	change.File = fa.File
	change.IsCreate = fa.IsCreate
	change.IsLocal = false
	change.PreviousHash = fa.PreviousHash
	return
}

// ah shoot! is this the only function that needs to know about the last fileaction ID state?
// the following function is hypothetical and non-functional as of now

// func serverActions(UDPPing <-chan bool, fileActionId int64) (out chan structs.StateChange,
// 	errorChan chan interface{}, err error) {
// 	go func() {
// 		for {
// 			<-UDPPing
// 			// these return values are obviously wrong right now
// 			fileActions, id, err := client.DownloadClientFileActions(fileActionId)

// 			// need to rethink errors, assumption of a statechange is invalid
// 			if err != nil {
// 				writeError(err, structs.StateChange{}, "serverActions")
// 			}
// 			for _, fileAction := range fileActions {
// 				change := createServerStateChange(fileAction)
// 				out <- change
// 			}

// 		}
// 	}()
// 	return
// }

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

func makeFileAction(change structs.StateChange) (fa structs.FileAction) {
	fa.IsCreate = change.IsCreate
	fa.CreatedAt = change.File.CreatedAt
	fa.FileId = change.File.Id
	fa.File = change.File
	fa.PreviousHash = change.PreviousHash
	return
}

func fileActionSender(change structs.StateChange) {
	select {
	case <-change.Quit:
		gracefulQuit(change)
		return
	default:
		fileActions := make([]structs.FileAction, 1)
		fileActions[0] = makeFileAction(change)

		needed, err := client.SendFileActionsToServer(fileActions)
		if err != nil {
			writeError(err, change, "fileActionSender")
			return
		}
		// if len needed 0, need to do cleanup by signaling done
		if len(needed) == 0 {
			writeDone(change, fileActions[0])
			return
		}
		for _, need := range needed {
			go uploader(need, change, fileActions[0])
		}
	}
	return
}

func uploader(path string, change structs.StateChange, fa structs.FileAction) {
	select {
	case <-change.Quit:
		gracefulQuit(change)
		return
	default:
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			writeError(err, change, "uploader")
			return
		}
		err = client.UploadFileToServer(buf)
		if err != nil {
			writeError(err, change, "uploader")
			return
		}
		change.Done <- fa
		close(change.Error)
	}
	return
}

func gracefulQuit(change structs.StateChange) {
	close(change.Done)
	close(change.Error)
}

func hasher(change structs.StateChange) {
	select {
	case <-change.Quit:
		gracefulQuit(change)
		return
	default:
		h, err := getSha256FromFilename(change.File.Path)
		if err != nil {
			writeError(err, change, "hasher")
			return
		}
		change.File.Hash = h
		go fileActionSender(change)
	}
	return
}

// potential for deadly embrace when stephen tries to send a new channel while this func
// blocks on sending on out

// fix was to add a go func to write to out. I don't love this solution becuase it makes order
// of stuff coming out of the out channel non-deterministic, so any other ideas are invited.
func arbitraryFanIn(newChannels <-chan chan interface{}, out chan<- interface{}, removeOnEvent bool) {
	go func() {
		var ch <-chan interface{}
		chans := make([]reflect.SelectCase, 0)
		timeout := time.Tick(10 * time.Millisecond)
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
					continue
				}
				if removeOnEvent || !ok {
					lastIndex := len(chans) - 1
					chans[chosen] = chans[lastIndex]
					chans = chans[:lastIndex]
				}
				if ok {
					go func() { out <- value.Interface() }()
				}
			}
		}
	}()
}

func deleter(change structs.StateChange) {
	fi, err := os.Stat(change.File.Path)
	if err != nil {
		if os.IsNotExist(err) {
			writeDone(change, makeFileAction(change))
			return
		}
		writeError(err, change, "deleter")
		return
	}
	err = os.Remove(change.File.Path)
	if err != nil {
		writeError(err, change, "deleter")
		return
	}
	writeDone(change, makeFileAction(change))
}

func stephen(dataPath string, stateChanges <-chan structs.StateChange) {
	// spin up a goroutine that will fan in error messages using reflect.select
	// hand it an error channel, and add this to the main select statement
	// do the same thing for done, so I can write a generic fan-n-in function
	fileSystemState, err := fetchFileSystemState(dataPath)
	if err != nil {
		panic("Could not properly retrieve fileSystemState")
	}
	quitChannels := make(map[string]chan bool)
	newErrors := make((chan (chan interface{})))
	newDones := make((chan (chan interface{})))
	errors := make(chan interface{})
	dones := make(chan interface{})
	go arbitraryFanIn(newErrors, errors, true)
	go arbitraryFanIn(newDones, dones, true)
	for {
		// maybe it would be better to combine errors and dones into the same multiplexor?
		select {
		case e := <-errors:
			msg := e.(structs.ErrorMessage)
			fmt.Println("Experienced an error with ", msg.File.Path)
			fmt.Println("In function: ", msg.Function)
			fmt.Println("Error: ", msg.Error)
			fmt.Println("File: ", msg.File)
			delete(quitChannels, msg.File.Path)
		case d := <-dones:
			fa := d.(structs.FileAction)
			delete(quitChannels, fa.File.Path)
			fileSystemState.FileActionId = fa.Id
			fileSystemState.State[fa.File.Path] = fa.File
		case change := <-stateChanges:
			if ch, found := quitChannels[change.File.Path]; found {
				// tell goroutine branch to quit
				ch <- true
				close(ch)
			}
			f, found := fileSystemState.State[change.File.Path]
			if change.IsLocal {
				if found {
					change.PreviousHash = f.Hash
				} else {
					change.PreviousHash = ""
				}
			}

			fmt.Println(change)
			quitChan := make(chan bool, 1)
			doneChan := make(chan interface{}, 1)
			newDones <- doneChan
			errChan := make(chan interface{}, 1)
			newErrors <- errChan
			quitChannels[change.File.Path] = quitChan
			change.Quit = quitChan
			change.Done = doneChan
			change.Error = errChan
			if change.IsCreate {
				if change.IsLocal {
					go hasher(change)
				} else {
					go downloader(change)
				}
			} else {
				if found {
					if f.Hash == change.PreviousHash {
						go deleter(change)
						delete(fileSystemState.State, change.File.Path)
					}
				}
			}

		}
	}
	fmt.Println(fileSystemState)
}

func run(path string) {
	go func() {
		client = api.New("foo")

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

	}()
}

func main() {
	run("../sandbox")
}
