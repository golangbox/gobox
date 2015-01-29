package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type file struct {
	name string
	hash string
	size int64
}

type fileInfo struct {
	Name     string
	Hash     string
	Size     int64
	Path     string
	Modified time.Time
}

var f1 = file{"file1", "sjdalkjsda", 1234}
var f2 = file{"file2", "lkjlkjlkl", 4321}
var allFiles = map[string]file{"file1": f1, "file2": f2}

//Compile templates on start
var templates = template.Must(template.ParseFiles("templates/upload.html"))

//Display the named template
func display(w http.ResponseWriter, tmpl string, data interface{}) {
	templates.ExecuteTemplate(w, tmpl+".html", data)
}

func fileListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(allFiles)
}

//This is where the action happens.
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	//GET displays the upload form.
	case "GET":
		log.Println("Get...")
		display(w, "upload", nil)

	//POST takes the uploaded file(s) and saves it to disk.
	case "POST":
		//get the multipart reader for the request.
		reader, err := r.MultipartReader()

		log.Println("Receiving stuff...")
		//hash := r.FormValue("sha-1")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//copy each part to destination.
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			//if part.FileName() is empty, skip this iteration.
			if part.FileName() == "" {
				continue
			}

			dst, err := os.Create("/Users/partec/hackerschool/go/gobox/tmp/" + part.FileName())
			defer dst.Close()

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if _, err := io.Copy(dst, part); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		//display success message.
		display(w, "upload", "Upload successful.")
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeLocalMeta(fileMeta []byte) {
	f, err := os.OpenFile("localMeta.gob", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(string(fileMeta) + "\n"); err != nil {
		panic(err)
	}
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

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)

	}
}

// filename, size, sha-1

func main() {
	fmt.Println("Server up...")
	http.HandleFunc("/upload", uploadHandler)

	http.HandleFunc("/files", fileListHandler)

	http.HandleFunc("/meta", handleMetaConnection)

	//static file handler.
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	//Listen on port 8080
	http.ListenAndServe(":4243", nil)
}
