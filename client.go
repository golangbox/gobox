package main

import (
	"bytes"
	"container/heap"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	kGoBoxDirectory     = "."
	kGoBoxDataDirectory = ".GoBox"
)

type FileInfo struct {
	Name     string
	Hash     string
	Size     int64
	Path     string
	Modified time.Time
}

type UploadInfo struct {
	Task string
	File FileInfo
}

type UploadHeap []UploadInfo

func (h UploadHeap) Len() int           { return len(h) }
func (h UploadHeap) Less(i, j int) bool { return 0 > h[i].File.Modified.Sub(h[j].File.Modified) }
func (h UploadHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *UploadHeap) Push(x interface{}) {

	*h = append(*h, x.(UploadInfo))
}

func (h *UploadHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

var uploadQueue *UploadHeap = &UploadHeap{}
var uploading bool = false

func main() {
	fmt.Println("Running GoBox...")
	// upload_file(name)
	createGoBoxLocalDirectory()

	heap.Init(uploadQueue)

	go monitorFiles()

	go watchAndProcessUploadQueue()

	select {}

}

func watchAndProcessUploadQueue() {
	for _ = range time.Tick(10 * time.Second) {
		fmt.Println("Checking to see if there's anything to upload.")
		if !uploading && uploadQueue.Len() > 0 {
			fmt.Println("There are things to upload and we're not already uploading. Let's upload!")
			uploading = true
			for uploadQueue.Len() > 0 {
				popped := heap.Pop(uploadQueue).(UploadInfo)
				fmt.Println("Uploading task:", popped.Task, popped.File.Name)
				time.Sleep(time.Second * 60)
			}
			uploading = false
		}
	}
}

func createGoBoxLocalDirectory() {

	_, err := os.Stat(kGoBoxDataDirectory)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Making directory")
		err := os.Mkdir(kGoBoxDataDirectory, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func monitorFiles() {

	var newFileInfos map[string]FileInfo = make(map[string]FileInfo)
	var fileInfos map[string]FileInfo = make(map[string]FileInfo)

	data, err := ioutil.ReadFile(kGoBoxDataDirectory + "/data")
	if err != nil {
		fmt.Println(err)
	}
	if data != nil {
		err = json.Unmarshal(data, &fileInfos)
		if err != nil {
			fmt.Println(err)
		}
	}

	for _ = range time.Tick(10 * time.Second) {
		fmt.Println("Checking to see if there are any filesystem changes.")
		newFileInfos, err = findFilesInDirectory(kGoBoxDirectory)
		if err != nil {
			fmt.Println(err)
		}

		err = writeFileInfosToLocalFile(newFileInfos)
		if err != nil {
			fmt.Println(err)
		} else {
			// fmt.Println("Saved data locally")
		}

		compareFileInfos(fileInfos, newFileInfos)

		fileInfos = newFileInfos
	}
}

func writeFileInfosToLocalFile(fileInfos map[string]FileInfo) error {
	jsonBytes, err := json.Marshal(fileInfos)
	if err != nil {
		return err
	}
	// d1 := []byte("hello\ngo\n")
	err = ioutil.WriteFile(kGoBoxDataDirectory+"/data", jsonBytes, 0644)
	return err
}

func compareFileInfos(fileInfos map[string]FileInfo, newFileInfos map[string]FileInfo) {

	for key, value := range newFileInfos {
		// http://stackoverflow.com/questions/2050391/how-to-test-key-existence-in-a-map
		if _, exists := fileInfos[key]; !exists {
			fmt.Println("Need to Upload:", key)
			heap.Push(uploadQueue, UploadInfo{"Upload", value})
		}
	}
	for key, value := range fileInfos {
		if _, exists := newFileInfos[key]; !exists {
			fmt.Println("Need to delete:", key)
			heap.Push(uploadQueue, UploadInfo{"Delete", value})
		}
	}
}

// http://matt.aimonetti.net/posts/2013/07/01/golang-multipart-file-upload-example/
func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
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

func getSha256FromFilename(filename string) (sha256_string string, err error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Error reading file for sha256: %s", err)
	}
	h := sha256.New()
	_, err = h.Write(file)
	if err != nil {
		return "", fmt.Errorf("Error writing file to hash for sha256: %s", err)
	}
	byte_string := h.Sum(nil)

	sha256_string = hex.EncodeToString(byte_string)

	return sha256_string, nil
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
	//"http://10.0.7.205:8080/upload"
	request, err := newfileUploadRequest("http://10.0.7.205:8080/upload", extraParams, "FileName", name)
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

func findFilesInDirectoryHelper(directory string, fileInfos map[string]FileInfo) (outputFileInfos map[string]FileInfo, err error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("Unable to read directory: %s", err)
	}

	for _, f := range files {
		name := f.Name()
		path := directory + "/" + name

		// fmt.Println("Scanning file: " + path)
		if f.IsDir() {
			if f.Name() != kGoBoxDataDirectory {
				fileInfos, err = findFilesInDirectoryHelper(path, fileInfos)
			}
		} else {
			sha256, err := getSha256FromFilename(path)
			if err != nil {
				fmt.Println(err)
			}

			fileInfos[mapKeyValue(path, sha256)] = FileInfo{name, sha256, f.Size(), path, f.ModTime()}
		}
	}

	return fileInfos, err
}

func findFilesInDirectory(directory string) (outputFileInfos map[string]FileInfo, err error) {
	var emptyFileInfos map[string]FileInfo = make(map[string]FileInfo)
	return findFilesInDirectoryHelper(directory, emptyFileInfos)
}
