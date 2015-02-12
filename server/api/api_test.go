package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/server/model"
	"github.com/golangbox/gobox/server/s3"
	"github.com/golangbox/gobox/structs"
	"github.com/jinzhu/gorm"
)

var user structs.User
var client structs.Client

func init() {
	model.DB, _ = gorm.Open("postgres", "dbname=goboxtest sslmode=disable")

	model.DB.DropTableIfExists(&structs.User{})
	model.DB.DropTableIfExists(&structs.Client{})
	model.DB.DropTableIfExists(&structs.FileAction{})
	model.DB.DropTableIfExists(&structs.File{})
	model.DB.DropTableIfExists(&structs.FileSystemFile{})
	model.DB.AutoMigrate(&structs.User{}, &structs.Client{}, &structs.FileAction{}, &structs.File{}, &structs.FileSystemFile{})

	user, _ = boxtools.NewUser("max.t.mcdonnell@gmail", "password")

	var err error
	client, err = boxtools.NewClient(user, "test", false)
	if err != nil {
		fmt.Println(err)
	}

	go ServeServerRoutes("8000")
}

// func httpErrorCheck(err error, statusCode int, w http.ResponseWriter)

func TestClientsFileActionsHandler(t *testing.T) {
	_, _ = boxtools.NewClient(user, "test", false)
	fileActions, _ := boxtools.GenerateSliceOfRandomFileActions(1, 1, 10)
	for _, value := range fileActions {
		model.DB.Create(&value)
	}
	resp, _ := http.PostForm(
		"http://localhost:8000/clients/",
		url.Values{"sessionKey": {client.SessionKey}},
	)
	contents, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fail()
	}
	var incomingFileActions []structs.FileAction
	json.Unmarshal(contents, &incomingFileActions)
	if len(incomingFileActions) != 10 {
		t.Fail()
	}
}

func TestUploadHandler(t *testing.T) {
	file := []byte("These are the file contents")
	resp, _ := http.Post(
		"http://localhost:8000/upload/?sessionKey="+client.SessionKey,
		"",
		bytes.NewBuffer(file),
	)
	contents, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		err := fmt.Errorf(string(contents))
		t.Error(err)
	}
	h := sha256.New()
	h.Write(file)
	byteString := h.Sum(nil)
	sha256String := hex.EncodeToString(byteString)

	url, _ := s3.GenerateSignedUrl(sha256String)

	resp, _ = http.Get(url)

	contents, _ = ioutil.ReadAll(resp.Body)

	if string(contents) != string(file) {
		t.Fail()
	}
}

func TestApiCallWithWrongAndNoAuth(t *testing.T) {
	resp, err := http.PostForm(
		"http://localhost:8000/file-actions/",
		url.Values{"sessionKey": {"nope"}},
	)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != 401 {
		t.Error("Wrong status code")
	}

	resp, err = http.Post(
		"http://localhost:8000/file-actions/",
		"text",
		bytes.NewBuffer([]byte("wheee")),
	)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != 401 {
		t.Error("Wrong status code")
	}
}

func TestFileActionsHandler(t *testing.T) {
	fileActions, _ := boxtools.GenerateSliceOfRandomFileActions(1, 1, 10)
	var bothfileActions []structs.FileAction
	for _, value := range fileActions {
		bothfileActions = append(bothfileActions, value)
		bothfileActions = append(bothfileActions, value)
	}
	jsonBytes, _ := json.Marshal(bothfileActions)
	resp, _ := http.Post(
		"http://localhost:8000/file-actions/?sessionKey="+client.SessionKey,
		"application/json",
		bytes.NewBuffer(jsonBytes),
	)
	contents, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fail()
	}
	var responseSlice []string
	json.Unmarshal(contents, &responseSlice)
	if len(responseSlice) != 10 {
		t.Fail()
	}
}

func TestFileDownloadHandler(t *testing.T) {
	file, _ := boxtools.GenerateRandomFile(1)
	model.DB.Create(&file)
	resp, err := http.PostForm(
		"http://localhost:8000/download/",
		url.Values{"sessionKey": {client.SessionKey}, "fileHash": {file.Hash}},
	)
	if err != nil {
		t.Error(err)
	}
	contents, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		// t.Fail()
	}
	url := string(contents)
	_ = url
	//not sure how we check to see if the url is valid
}
