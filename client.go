// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !plan9,!solaris

package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	// "encoding/json"
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
	goBoxDirectory = "."
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

func map_key_value(path string, sha1 string) (key string) {
	return path + "-" + sha1
}

func find_files(directory string, files_map map[string]Fileinfo) (output_files_map map[string]Fileinfo, err error) {
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
			key := map_key_value(path, sha1)
			files_map[key] = Fileinfo{name, sha1, f.Size(), path, f.ModTime()}
		} else {
			files_map, err = find_files(path, files_map)
		}
	}
	return files_map, err
}

func create_file_manifest(previous_manifest map[string]Fileinfo) (files_map map[string]Fileinfo, err error) {
	//get files and folders in current directory
	directory := goBoxDirectory
	var empty_files_map map[string]Fileinfo
	empty_files_map = make(map[string]Fileinfo)

	files_map, err = find_files(directory, empty_files_map)

	// Implement a map of path + sha1
	// When a new manifest is created compare it to the old manifest.
	// When there are new entires that don't exist in the old manifest,
	// assume we have to create a new file
	// Only check files that are modified after the date of the last
	// manifest, for a small speedup.

	// When there are entires in the old manifest that don't exist in the
	// current manifest, assume we have to delete a file

	// This ignore chmod changes, and other metadata changes.

	for key, _ := range files_map {
		// http://stackoverflow.com/questions/2050391/how-to-test-key-existence-in-a-map
		if _, ok := previous_manifest[key]; !ok {
			fmt.Println("Need to Upload:", key)
		}
	}
	for key, _ := range previous_manifest {
		if _, ok := files_map[key]; !ok {
			fmt.Println("Need to delete:", key)
		}
	}

	// fmt.Println(files_slice)
	// b, err := json.Marshal(files_slice)
	// if err != nil {
	// 	return nil, fmt.Errorf("Error with json Marshal of file slice: %s", err)
	// }

	return files_map, err
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

func new_upload_file() {
	// send a sha1
	// if they don't have it, upload the file
}

func upload_manifest() {

}

func main() {
	fmt.Println("Running GoBox...")
	// upload_file(name)
	var empty_manifest map[string]Fileinfo
	empty_manifest = make(map[string]Fileinfo)

	manifest_map, err := create_file_manifest(empty_manifest)
	if err != nil {
		fmt.Println(err)
	}
	if manifest_map != nil {
		// for key, value := range manifest_map {
		// 	fmt.Println("Key:", key, "Value:", value)
		// }
		fmt.Println("File manifest created")
	}

	check_for_files()

	// watcher, err := fsnotify.NewWatcher()

	// if err != nil {
	// 	log.Fatal(err)
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
	// 	log.Fatal(err)
	// }
	// <-done

}
