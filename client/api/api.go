package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/golangbox/gobox/structs"
)

const (
	apiEndpoint = "http://localhost:8000/"
)

type Api struct {
	sessionKey string
}

func New(sessionKey string) (c Api) {
	resp, err := http.Get(apiEndpoint + "login/")
	if err != nil {
		log.Fatal(err)
	}
	keyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	c.sessionKey = string(keyBytes)
	return
}

func (c *Api) apiRequest(endpoint string, body []byte,
	fileType string) (*http.Response, error) {
	return http.Post(
		apiEndpoint+endpoint+"/?sessionKey="+c.sessionKey,
		"application/json",
		bytes.NewBuffer(body),
	)
}

func (c *Api) SendFileActionsToServer(
	fileActions []structs.FileAction) (
	filesToUpload []string, err error) {

	jsonBytes, err := json.Marshal(fileActions)
	if err != nil {
		return
	}

	resp, err := c.apiRequest(
		"file-actions",
		jsonBytes,
		"application/json",
	)

	if err != nil {
		return
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(string(contents))
		return
	}

	err = json.Unmarshal(contents, &filesToUpload)
	if err != nil {
		return
	}
	return
}

func (c *Api) UploadFileToServer(fileBody []byte) (err error) {
	resp, err := c.apiRequest(
		"upload",
		fileBody,
		"",
	)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = fmt.Errorf(string(contents))
		return err
	}
	return
}

func (c *Api) DownloadFileFromServer(
	hash string) (s3_url string, err error) {
	resp, err := http.PostForm(
		apiEndpoint+"download/",
		url.Values{
			"sessionKey": {c.sessionKey},
			"fileHash":   {hash},
		},
	)
	if err != nil {
		return
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(string(contents))
		return
	}
	s3_url = string(contents)
	return
}

// this needs to return the highest FileActionID
func (c *Api) DownloadClientFileActions(
	lastId int) (fileActions []structs.FileAction, err error) {

	lastId = 0 // don't have support for this just yet
	var lastIdString string
	lastIdString = strconv.Itoa(lastId)
	resp, err := http.PostForm(
		apiEndpoint+"/clients/",
		url.Values{
			"sessionKey": {c.sessionKey},
			"lastID":     {lastIdString},
		},
	)
	if err != nil {
		return
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(string(contents))
		return
	}

	err = json.Unmarshal(contents, &fileActions)
	if err != nil {
		return
	}
	return
}
