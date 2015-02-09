package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

type uploadInfo struct {
	Task string
	File fileInfo
}
type fileInfo struct {
	Name     string
	Hash     string
	Size     int64
	Path     string
	Modified time.Time
}

// func writeLocalMeta(fileMeta []byte) {
// 	f, err := os.OpenFile("localMeta.gob",
// 		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
// 	if err != nil {
// 		panic(err)
// 	}

// 	defer f.Close()

// 	if _, err = f.WriteString(string(fileMeta) + "\n"); err != nil {
// 		panic(err)
// 	}
// }

// The high level view goes like this:

// client detects change, sends an http request on the /meta endpoint as a post

// we can call the Task "commit" or whatever.

// server must check to see if the hash already exists for that user in the "File" db table
// so that check is in the form of db.Where("user=? AND hash=?", user, hash).First(&file) (in pseudo-form)

// If so, write the metadata to the Journal in the form of Task_ID | File_ID

// if not, tell the client that it needs to upload to file with a json object

// the client uploads the file to the server (how will this be accomplished? https? sftp?) <- needs to be able to be cancelled
// in case the file is changed on the client while this process is ongoing (once the file has been uploaded to the server,
// everything is good)

// then the client attempts a second commit after successful upload
// when the server has the hash of the file, the server then adds an entry to the "journal" table

func evalAction(contents []byte) {
	var uploadData uploadInfo

	err := json.Unmarshal(contents, &uploadData)
	fileMeta := uploadData.File

	if err != nil {
		fmt.Errorf("Error: %s", err)
	}

	switch uploadData.Task {
	case "upload":
		fmt.Printf("[*]Uploading %s to S3...", fileMeta.Name)
	}
}

func authOnS3() {
	key := os.Getenv("GOBOX_AWS_ACCESS_KEY_ID")
	secret := os.Getenv("GOBOX_AWS_SECRET_ACCESS_KEY")

	auth, err := aws.GetAuth(key, secret)
	if err != nil {
		log.Fatal(err)
	}
	client := s3.New(auth, aws.USEast)
	resp, err := client.ListBuckets()

	if err != nil {
		log.Fatal(err)
	}

	log.Print(fmt.Sprintf("%T %+v", resp.Buckets[0], resp.Buckets[0]))
}

func handleMetaConnection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "GET":
		log.Println("[+] GET REQUEST RECEIVED")

	case "POST":
		log.Println("[+] POST REQUEST RECEIVED")
		contents, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic("Error")
		}
		writeLocalMeta(contents)
		evalAction(contents)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)

	}
}

func main() {

	//// Upload endpoint
	//http.HandleFunc("/upload", uploadHandler)

	//// files endpoint
	//http.HandleFunc("/files", fileListHandler)

	// meta endpoint
	http.HandleFunc("/meta", handleMetaConnection)

	//static file handler.
	http.Handle("/assets/",
		http.StripPrefix("/assets/",
			http.FileServer(http.Dir("assets"))))
	// authOnS3()
	//Listen
	fmt.Println("[+] Server Initialized on port: 4243")
	http.ListenAndServe(":4243", nil)

}
