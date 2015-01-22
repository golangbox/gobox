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

type Fileinfo struct {
	Name string
	Hash string
	Size int64
}

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

func file_sha1(name string) (byte_string []byte) {
	file, _ := ioutil.ReadFile(name)
	h := sha1.New()
	h.Write(file)
	byte_string = h.Sum(nil)
	return byte_string
}

// func print_file_info(name) {

// }

func upload_file() {
	filename := "main.go"
	file, _ := os.Stat(filename)
	bs := file_sha1(filename)
	s := hex.EncodeToString(bs)

	size := strconv.Itoa(int(file.Size()))
	extraParams := map[string]string{
		"Name": file.Name(),
		"Hash": s,
		"Size": size,
	}

	// http://requestb.in/19w82ne1
	//"http://10.0.7.205:8080/upload"
	request, err := newfileUploadRequest("http://10.0.7.205:8080/upload", extraParams, "FileName", "main.go")
	client := &http.Client{}
	fmt.Println(request)
	resp, err := client.Do(request)
	fmt.Println(resp)
	fmt.Println(request, err)

}

func main() {

	upload_file()

	file_slice := []Fileinfo{}

	files, _ := ioutil.ReadDir("./")
	for _, f := range files {
		// fmt.Println(f.Name(), f.Size(), f.Sys())
		fmt.Println(f.Name(), f.Size(), f.Mode(), f.IsDir())
		bs := file_sha1(f.Name())
		s := hex.EncodeToString(bs)
		fmt.Println(s)
		file_data := Fileinfo{f.Name(), s, f.Size()}
		// tmp_slice := []Fileinfo{file_data}
		file_slice = append(file_slice, file_data)
	}
	fmt.Println(file_slice)
	b, _ := json.Marshal(file_slice)
	fmt.Println(string(b))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				// f, _ := os.Stat(event.Name)
				// fmt.Println(f.Name(), f.Size(), f.Mode(), f.IsDir())
				bs := file_sha1(event.Name)
				fmt.Printf("%x\n", bs)
				// fmt.Println(string(sha1.Sum(file)))
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(".") // listen in the current directory
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
