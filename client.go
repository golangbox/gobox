// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !plan9,!solaris

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-fsnotify/fsnotify"
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

func main() {
	fmt.Println("Running GoBox...")
	// upload_file(name)
	createGoBoxLocalDirectory()

	go monitorFiles()

	select {}

	// watcher, err := fsnotify.NewWatcher()

	// if err != nil {
	//  log.Fatal(err)
	// }
	// defer watcher.Close()

	// done := make(chan bool)
	// fmt.Println("Listening for file changes...")
	// go watchFiles(watcher)
	// // directories = getDirsOnPWD
	// // go recursiveListeners(directories, channel)
	// // for event range channel { perform operations on event because it happened in some goroutine }
	// err = watcher.Add(".") // listen in the current directory
	// if err != nil {
	//  log.Fatal(err)
	// }
	// <-done

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

	// type manifest struct {
	// 		Map map
	// 		updateManifest() func
	// }

	// monitoring class
	// manifest object class
	//

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

		newFileInfos, err = findFilesInDirectory(kGoBoxDirectory)
		if err != nil {
			fmt.Println(err)
		}

		err = writeFileInfosToLocalFile(newFileInfos)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Saved data locally")
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

	for key, _ := range newFileInfos {
		// http://stackoverflow.com/questions/2050391/how-to-test-key-existence-in-a-map
		if _, exists := fileInfos[key]; !exists {
			fmt.Println("Need to Upload:", key)
		}
	}
	for key, _ := range fileInfos {
		if _, exists := newFileInfos[key]; !exists {
			fmt.Println("Need to delete:", key)
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

// http://jxck.hatenablog.com/entry/golang-error-handling-lesson-by-rob-pike
// type errWriter struct {
// 	w   io.Writer
// 	err error
// }

// func (e *errWriter) Write(p []byte) {
// 	if e.err != nil {
// 		return
// 	}
// 	_, e.err = e.w.Write(p)
// }

// func (e *errWriter) Err() error {
// 	return e.err
// }

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

type FileInfo struct {
	Name     string
	Hash     string
	Size     int64
	Path     string
	Modified time.Time
}

func watchFiles(watcher *fsnotify.Watcher) {
	// watches files in the current directory
	// this should potentially be used to listed for general changes on the
	// filesystem, but is too noisy to be used to calculate the index and
	// trigger uploads
	for {
		select {
		case event := <-watcher.Events:
			log.Println("event:", event)
			// f, _ := os.Stat(event.Name)
			// fmt.Println(f.Name(), f.Size(), f.Mode(), f.IsDir())
			sha256, err := getSha256FromFilename(event.Name)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(sha256)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}
