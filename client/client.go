package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/golangbox/gobox/client/api"
	"github.com/golangbox/gobox/client/watcher"
	"github.com/golangbox/gobox/structs"
)

// TODO
// get client/server sessions set up
// sync a damn file
// work on download branch

// PROBLEMS: No way to tell if a remove event was dir or a file because it can't be os.Stat'ed
//           Can't remove that dir from a watch because Watcher.watches isn't exposed

var client api.Api

const (
	dataDirectoryBasename = ".Gobox"
	serverEndpoint        = "http://requestb.in/1mv9fa41"
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

func findChangedFilesOnInit(goboxDirectoryPath string,
	fileSystemStatePath string) (err error) {
	fileSystemState, err := fetchFileSystemState(fileSystemStatePath)
	out := make(chan structs.StateChange)
	go func() {
		err = filepath.Walk(goboxDirectoryPath, func(fp string, fi os.FileInfo, errIn error) (errOut error) {

			if err != nil {
				panic("Couldn't read filesystem state during findChangedFilesOnInit")
			}
			matched, errOut := regexp.MatchString(".Gobox(/.*)*", fp)
			if errOut != nil {
				return
			}
			if matched {
				fmt.Println(fp)
				return
			}
			f, found := fileSystemState[fp]
			if !found {
				change, err := watcher.CreateLocalStateChange(fp, watcher.CREATE)
				if err != nil {
					return
				}
				out <- change
				return
			}
			h, err := getSha256FromFilename(fp)
			if err != nil {
				return
			}
			if f.Hash != h {
				change, err := watcher.CreateLocalStateChange(fp, watcher.MODIFY)
				if err != nil {
					return
				}
				out <- change
				delete(fileSystemState, fp)
				return
			}
			return
		})
		if err != nil {
			return
		}
		for fp, _ := range fileSystemState {
			// may need to change structs so that PreviousHash is in the File struct
			change, err := watcher.CreateLocalStateChange(fp, watcher.DELETE)
			if err != nil {
				return
			}
			out <- change
		}
		return
	}()
	return
}

func startWatcher(dir string) (out chan structs.StateChange, err error) {
	fmt.Println(dir)
	rw, err := watcher.NewRecursiveWatcher(dir)
	if err != nil {
		return out, err

	}
	rw.Run(false)
	return rw.Files, err
}

func createServerStateChange(fa structs.FileAction) (change structs.StateChange) {
	change.File = fa.File
	change.IsCreate = fa.IsCreate
	change.IsLocal = false
	change.PreviousHash = fa.PreviousHash
	return
}

// ah shoot! is this the only function that needs to know about the last fileaction ID state?
// the following function is hypothetical and non-functional as of now

func serverActions(UDPing <-chan bool, fileActionIdPath string) (out chan structs.StateChange,
	errorChan chan interface{}, err error) {
	fileActionId, err := fetchFileActionID(fileActionIdPath)
	if err != nil {
		panic("Couldn't properly read fileActionId from given path")
	}

	go func() {
		for {
			<-UDPing
			fmt.Println("-----------------PING RECIEVED--------------------")
			// these return values are obviously wrong right now
			clientFileActionResponse, err := client.DownloadClientFileActions(fileActionId)
			// need to rethink errors, assumption of a statechange is invalid
			if err != nil {
				writeError(err, structs.StateChange{}, "serverActions")
			}
			fileActionId = clientFileActionResponse.LastId
			for _, fileAction := range clientFileActionResponse.FileActions {
				change := createServerStateChange(fileAction)
				out <- change
			}
			err = writeFileActionIDToLocalFile(fileActionId, fileActionIdPath)
			if err != nil {
				fmt.Println("Couldn't write fileActionId to path : ", fileActionIdPath)
			}
		}
	}()
	return
}

func initUDPush(sessionKey string) (notification chan bool, err error) {
	go func() {
		conn, err := net.Dial("udp", api.UDPEndpoint)
		// defer conn.Close()
		if err != nil {
			return
		}
		sessionKeyBytes := []byte(sessionKey)
		_, err = conn.Write(sessionKeyBytes)
		if err != nil {
			return
		}
		response := make([]byte, 1)
		for {
			read, _ := conn.Read(response)
			fmt.Println(read)
			notification <- true
		}
	}()
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

func writeFileSystemStateToLocalFile(fileSystemState map[string]structs.File, path string) error {
	jsonBytes, err := json.Marshal(fileSystemState)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, jsonBytes, 0644)
	return err
}

func fetchFileSystemState(path string) (fileSystemState map[string]structs.File, err error) {
	if _, err := os.Stat(path); err != nil {
		fmt.Println("Making empty data file")
		emptyState := make(map[string]structs.File)
		writeFileSystemStateToLocalFile(emptyState, path)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if data != nil {
		err = json.Unmarshal(data, &fileSystemState)
	}
	return

}

func writeFileActionIDToLocalFile(id int64, path string) (err error) {
	jsonBytes, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, jsonBytes, 0644)
	return
}

func fetchFileActionID(path string) (id int64, err error) {
	if _, err := os.Stat(path); err != nil {
		defaultId := int64(1)
		err = writeFileActionIDToLocalFile(defaultId, path)
		if err != nil {
			return id, err
		}
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	if data != nil {
		err = json.Unmarshal(data, &id)
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
		// need to fix this to just get responses for one file
		go uploader(change.File.Path, change, fileActions[0])

	}
	return
}

func uploader(path string, change structs.StateChange, fa structs.FileAction) {
	select {
	case <-change.Quit:
		gracefulQuit(change)
		return
	default:
		fmt.Println(path)
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

func serverDeleter(change structs.StateChange) {
	_, err := os.Stat(change.File.Path)
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

func downloader(change structs.StateChange) {
	select {
	case <-change.Quit:
		gracefulQuit(change)
		return
	default:
		// this could take a long time
		s3_url, err := client.DownloadFileFromServer(change.File.Hash)
		if err != nil {
			writeError(err, change, "downloader")
		}
		resp, err := http.Get(s3_url)
		if err != nil {
			writeError(err, change, "downloader")
		}
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			writeError(err, change, "downloader")
		}
		tmpFilename := filepath.Join(".Gobox/tmp/", change.File.Hash)
		err = ioutil.WriteFile(tmpFilename, contents, 0644)
		if err != nil {
			writeError(err, change, "downloader")
		}
		select {
		case <-change.Quit:
			gracefulQuit(change)
			return
		default:
			err = os.Rename(tmpFilename, change.File.Path)
			if err != nil {
				writeError(err, change, "downloader")
			}
		}
	}
	return
}

func localDeleter(change structs.StateChange) {
	go fileActionSender(change)
}

func stephen(goboxFileSystemStateFile string, stateChanges <-chan structs.StateChange,
	inputErrChans []chan interface{}) {
	// spin up a goroutine that will fan in error messages using reflect.select
	// hand it an error channel, and add this to the main select statement
	// do the same thing for done, so I can write a generic fan-n-in function
	fileSystemState, err := fetchFileSystemState(goboxFileSystemStateFile)
	if err != nil {
		panic("Could not properly retrieve fileSystemState")
	}

	quitChannels := make(map[string]structs.CurrentAction)

	newErrors := make((chan (chan interface{})))
	errors := make(chan interface{})
	go arbitraryFanIn(newErrors, errors, true)

	newDones := make((chan (chan interface{})))
	dones := make(chan interface{})
	go arbitraryFanIn(newDones, dones, true)

	for _, ch := range inputErrChans {
		newErrors <- ch
	}

	writeFileSystemStateCounter := 0
	for {
		if writeFileSystemStateCounter > 5 {
			writeFileSystemStateToLocalFile(
				fileSystemState,
				goboxFileSystemStateFile,
			)
			if err != nil {
				fmt.Println(
					"Error while writing fileSystemState to: ",
					goboxFileSystemStateFile)
			}
			writeFileSystemStateCounter = 0

		}
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
			fileSystemState[fa.File.Path] = fa.File
		case change := <-stateChanges:
			if currentAction, found := quitChannels[change.File.Path]; found {
				// tell goroutine branch to quit
				currentAction.Quit <- true
				close(currentAction.Quit)
			}
			f, found := fileSystemState[change.File.Path]
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
			quitChannels[change.File.Path] = structs.CurrentAction{Quit: quitChan, IsCreate: change.IsCreate}
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
					if change.IsLocal {
						go localDeleter(change)
					} else {
						if f.Hash == change.PreviousHash {
							go serverDeleter(change)
							delete(fileSystemState, change.File.Path)
						}

					}
				}

			}
		}
		writeFileSystemStateCounter++
	}
}
func run(path string) {
	go func() {
		errChans := make([]chan interface{}, 0)
		client = api.New("")
		fmt.Println(client.SessionKey)
		err := os.Chdir(path)
		if err != nil {
			fmt.Println("unable to change dir, quitting")
			return
		}
		goboxDirectory := "."
		goboxDataDirectory := filepath.Join(goboxDirectory, dataDirectoryBasename)
		goboxFileSystemStateFile := filepath.Join(goboxDataDirectory, "fileSystemState")
		goboxFileActionIdFile := filepath.Join(goboxDataDirectory, "fileActionId")

		createGoboxLocalDirectory(goboxDataDirectory)
		// findChangedFilesOnInit(goboxDirectory, goboxFileSystemStateFile)
		watcherActions, err := startWatcher(goboxDirectory)
		if err != nil {
			panic("Could not start watcher")
		}
		UDPNotification, err := initUDPush(client.SessionKey)
		if err != nil {
			panic("Could not start UDP socket")
		}
		// fix this to get correct fileActionID
		remoteActions, errChan, err := serverActions(UDPNotification, goboxFileActionIdFile)
		errChans = append(errChans, errChan)
		if err != nil {
			panic("Could not properly start remote actions")
		}
		actions := fanActionsIn(watcherActions, remoteActions)

		stephen(goboxFileSystemStateFile, actions, errChans)

		fmt.Println(watcherActions)
		for {
			time.Sleep(1000)
		}

	}()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: ./gobox_client PATH_TO_GOBOX_DIRECTORY")
		return
	}
	fi, err := os.Stat(os.Args[1])
	if err != nil {
		fmt.Println("Error reading gobox directory")
		return
	}
	if !fi.IsDir() {
		fmt.Println("Provided path is not a directory")
		return
	}

	fmt.Println("Running : ", os.Args[1])
	run(os.Args[1])
	for {
		time.Sleep(1000)
	}
}
