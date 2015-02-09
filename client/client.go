package main

import (
	"bytes"
	"container/heap"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gobox/boxtools"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	goBoxDirectory     = "."
	goBoxDataDirectory = ".GoBox"
	serverEndpoint     = "http://localhost:4243"
	// serverEndpoint           = "http://www.google.com"
	filesystemCheckFrequency = 5
)

type uploadHeap []boxtools.UploadInfo

func (h uploadHeap) Len() int           { return len(h) }
func (h uploadHeap) Less(i, j int) bool { return 0 > h[i].File.Modified.Sub(h[j].File.Modified) }
func (h uploadHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *uploadHeap) Push(x interface{}) {

	*h = append(*h, x.(boxtools.UploadInfo))
}

func (h *uploadHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

var uploadQueue = &uploadHeap{}
var uploading = false

func main() {
	fmt.Println("Running GoBox...")
	// upload_file(name)
	createGoBoxLocalDirectory()

	heap.Init(uploadQueue)

	go monitorFiles()

	// go watchAndProcessUploadQueue()

	select {}

}

func watchAndProcessUploadQueue() {
	for _ = range time.Tick(filesystemCheckFrequency * time.Second) {
		// fmt.Println("Checking to see if there's anything to upload.")
		if !uploading && uploadQueue.Len() > 0 {
			// fmt.Println("There are things to upload and we're not already uploading. Let's upload!")
			uploading = true
			for uploadQueue.Len() > 0 {
				poppedFileInfo := heap.Pop(uploadQueue).(boxtools.UploadInfo)
				_ = poppedFileInfo
				// resp, err := uploadMetadata(poppedFileInfo)
				// fmt.Println(resp, err)
			}
			uploading = false
		}
	}
}

func createGoBoxLocalDirectory() {

	_, err := os.Stat(goBoxDataDirectory)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Making directory")
		err := os.Mkdir(goBoxDataDirectory, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func monitorFiles() {

	var newfileInfos = make(map[string]boxtools.FileInfo)
	var fileInfos = make(map[string]boxtools.FileInfo)

	data, err := ioutil.ReadFile(goBoxDataDirectory + "/data")
	if err != nil {
		fmt.Println(err)
	}
	if data != nil {
		err = json.Unmarshal(data, &fileInfos)
		if err != nil {
			fmt.Println(err)
			//asdfsda
		}
	}

	// lastFilesystemCheck := make(chan time.Time, 1)

	for _ = range time.Tick(filesystemCheckFrequency * time.Second) {
		fmt.Println("Checking to see if there are any filesystem changes.")
		newfileInfos, err = findFilesInDirectory(goBoxDirectory)
		if err != nil {
			fmt.Println(err)
		}
		err := comparefileInfos(fileInfos, newfileInfos)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Error uploading file changes, skipping this upload cycle")
		} else {
			//sadfasdfasdfsa
			err = writefileInfosToLocalFile(newfileInfos)
			if err != nil {
				fmt.Println(err)
			} else {
				// fmt.Println("Saved data locally")
			}
			fileInfos = newfileInfos
		}
	}
}

func writefileInfosToLocalFile(fileInfos map[string]boxtools.FileInfo) error {
	jsonBytes, err := json.Marshal(fileInfos)
	if err != nil {
		return err
	}
	// d1 := []byte("hello\ngo\n")
	err = ioutil.WriteFile(goBoxDataDirectory+"/data", jsonBytes, 0644)
	return err
}

func handleFileChange(action string, file boxtools.FileInfo) (err error) {
	infoToSend := boxtools.UploadInfo{action, file}
	heap.Push(uploadQueue, infoToSend)
	resp, err := uploadMetadata(infoToSend)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error uploading metadata, status code: %d", resp.StatusCode)
		return
	}

	// resp, err :=
	return err
	// fmt.Println(resp)
	//asdfsdaf
}

func uploadMetadata(uploadinfo boxtools.UploadInfo) (resp *http.Response, err error) {
	fmt.Println("Uploading metadata: ("+uploadinfo.Task, uploadinfo.File.Name+")")
	jsonBytes, err := json.Marshal(uploadinfo)
	if err != nil {
		return
	}
	resp, err = http.Post(serverEndpoint+"/meta", "application/json", bytes.NewBuffer(jsonBytes))
	fmt.Println(resp)
	if uploadinfo.Task == "upload" {
		var contents []byte
		contents, err = ioutil.ReadAll(resp.Body)
		err := resp.Body.Close()
		if err != nil {
			return
		}
		fmt.Println(string(contents))
		if string(contents) == "true" {
			resp, err = uploadFile(uploadinfo.File.Path)
			if err != nil {
				return
			}
		} else {
			fmt.Println("no need for upload")
		}
	}
	if err != nil {
		return
	}
	return
}

func comparefileInfos(fileInfos map[string]boxtools.FileInfo,
	newfileInfos map[string]boxtools.FileInfo) (err error) {

	for key, value := range newfileInfos {
		// http://stackoverflow.com/questions/2050391/how-to-test-key-existence-in-a-map
		if _, exists := fileInfos[key]; !exists {
			fmt.Println("Need to Upload:", value.Name)
			err = handleFileChange("upload", value)
			if err != nil {
				return err
			}
		}
	}
	for key, value := range fileInfos {
		if _, exists := newfileInfos[key]; !exists {
			fmt.Println("Need to delete:", key)
			err = handleFileChange("delete", value)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// http://matt.aimonetti.net/posts/2013/07/01/golang-multipart-file-upload-example/
func newfileUploadRequest(uri string, params map[string]string,
	paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	return req, err
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

func uploadFile(name string) (*http.Response, error) {
	// filename := "main.go"
	file, _ := os.Stat(name)
	s, err := getSha256FromFilename(name)
	if err != nil {
		fmt.Println(err)
	}

	size := strconv.Itoa(int(file.Size()))
	extraParams := map[string]string{
		"Name": file.Name(),
		"Hash": s,
		"Size": size,
	}

	// http://requestb.in/19w82ne1
	//"http://10.0.7.205:4242/upload"
	request, err :=
		newfileUploadRequest("http://10.0.7.205:4242/upload",
			extraParams, "FileName", name)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	// resp, err := client.Do(request)
	return client.Do(request)
}

func mapKeyValue(path string, sha256 string) (key string) {
	return path + "-" + sha256
}

func findFilesInDirectoryHelper(directory string,
	fileInfos map[string]boxtools.FileInfo) (outputfileInfos map[string]boxtools.FileInfo,
	err error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("Unable to read directory: %s", err)
	}

	for _, f := range files {
		name := f.Name()
		path := directory + "/" + name

		// fmt.Println("Scanning file: " + path)
		if f.IsDir() {
			if f.Name() != goBoxDataDirectory {
				fileInfos, err = findFilesInDirectoryHelper(path, fileInfos)
			}
		} else {
			sha256, err := getSha256FromFilename(path)
			if err != nil {
				fmt.Println(err)
			}

			fileInfos[mapKeyValue(path, sha256)] = boxtools.FileInfo{name,
				sha256, f.Size(), path, f.ModTime()}
		}
	}

	return fileInfos, err
}

func findFilesInDirectory(directory string) (outputfileInfos map[string]boxtools.FileInfo,
	err error) {
	var emptyfileInfos = make(map[string]boxtools.FileInfo)
	return findFilesInDirectoryHelper(directory, emptyfileInfos)
}
