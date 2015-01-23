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
		return "", fmt.Errorf("Error reading file: %s", err)
	}

	h := sha1.New()

	_, err = h.Write(file)

	if err != nil {
		return "", fmt.Errorf("Error writing file to hash: %s", err)
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

func create_file_manifest() {
	files, err := ioutil.ReadDir("./") //read all the files
	if err != nil {
		fmt.Println(fmt.Errorf("Unable to read directory: %s", err))
	}

	files_slice := []Fileinfo{}
	for _, f := range files {
		// fmt.Println(f.Name(), f.Size(), f.Sys())
		fmt.Println(f.Name(), f.Size(), f.Mode(), f.IsDir())

		s, err := file_sha1(f.Name())
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(s)
		file_data := Fileinfo{f.Name(), s, f.Size()}
		// tmp_slice := []Fileinfo{file_data}
		files_slice = append(files_slice, file_data)
	}
	fmt.Println(files_slice)
	b, _ := json.Marshal(files_slice)
	fmt.Println(string(b))
}

type Fileinfo struct {
	Name string
	Hash string
	Size int64
}

func watchFiles(watcher *fsnotify.Watcher) {
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

	// upload_file(name)
	create_file_manifest()

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go watchFiles(watcher)
	// directories = getDirsOnPWD
	// go recursiveListeners(directories, channel)
	// for event range channel { perform operations on event because it happened in some goroutine }
	err = watcher.Add(".") // listen in the current directory
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
