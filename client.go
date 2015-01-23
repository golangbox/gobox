// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !plan9,!solaris

package main

import (
	"bytes"
	"crypto/sha1"
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

func file_sha1(name string) (sha1_string string, err error) {
	file, err := ioutil.ReadFile(name)

	if err != nil {
		return "", fmt.Errorf("Error reading file for sha1: %s", err)
	}

	h := sha1.New()

	_, err = h.Write(file)

	if err != nil {
		return "", fmt.Errorf("Error writing file to hash for sha1: %s", err)
	}

	byte_string := h.Sum(nil)

	hex.EncodeToString(byte_string)
	sha1_string = hex.EncodeToString(byte_string)

	return sha1_string, nil
}

func upload_file(name string) (*http.Response, error) {
	// filename := "main.go"
	file, _ := os.Stat(name)
	s, err := file_sha1(name)
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

func find_files(directory string, files_slice []Fileinfo) (output_files_slice []Fileinfo, err error) {
	files, err := ioutil.ReadDir(directory)

	if err != nil {
		return nil, fmt.Errorf("Unable to read directory: %s", err)
	}
	for _, f := range files {
		name := f.Name()
		path := directory + "/" + name
		fmt.Println("Scanning file: " + path)
		if !f.IsDir() { // if file is not directory

			// fmt.Println(f.Name(), f.Size(), f.Mode(), f.IsDir(), f.Sys())

			sha1, err := file_sha1(path)
			if err != nil {
				fmt.Println(err)
			}
			// fmt.Println(s)
			file_data := Fileinfo{name, sha1, f.Size(), path, f.ModTime()}
			files_slice = append(files_slice, file_data)
		} else {
			files_slice, err = find_files(path, files_slice)
		}
	}
	return files_slice, err
}

func create_file_manifest() (json_bytes []byte, err error) {
	//get files and folders in current directory
	directory := "."
	files_slice := []Fileinfo{}

	files_slice, err = find_files(directory, files_slice)

	// fmt.Println(files_slice)
	b, err := json.Marshal(files_slice)
	if err != nil {
		return nil, fmt.Errorf("Error with json Marshal of file slice: %s", err)
	}

	return b, err
}

type Fileinfo struct {
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
			s, err := file_sha1(event.Name)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(s)
			// fmt.Println(string(sha1.Sum(file)))
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func main() {
	fmt.Println("Running GoBox...")
	// upload_file(name)
	manifest, err := create_file_manifest()
	if err != nil {
		fmt.Println(err)
	}
	if manifest != nil {
		fmt.Println(string(manifest))
		fmt.Println("File manifest created")
	}

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// done := make(chan bool)
	// fmt.Println("Listening for file changes...")
	// go watchFiles(watcher)
	// // directories = getDirsOnPWD
	// // go recursiveListeners(directories, channel)
	// // for event range channel { perform operations on event because it happened in some goroutine }
	// err = watcher.Add(".") // listen in the current directory
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// <-done
}
