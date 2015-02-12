package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golangbox/gobox/structs"
)

const (
	apiEndpoint = "http://localhost:8000/"
)

type client struct {
	sessionKey string
}

func New(sessionKey string) (c client) {
	c.sessionKey = sessionKey
	return
}

func (c *client) apiRequest(endpoint string, body []byte,
	fileType string) (*http.Response, error) {
	return http.Post(
		apiEndpoint+endpoint+"/?sessionKey="+c.sessionKey,
		"application/json",
		bytes.NewBuffer(jsonBytes),
	)
}

func (c *client) SendFileActionsToServer(
	fileActions []structs.FileActions) (
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

	if resp.StatusCode != htt.StatusOK {
		err = fmt.Errorf(contents)
		return
	}

	err = json.Unmarshal(contents, &filesToUpload)
	if err != nil {
		return
	}
	return
}

func (c *client) UploadFileToServer(fileBody []byte) (err error) {
	resp, err = c.apiRequest(
		"upload",
		fileBody,
		"",
	)
	if err != nil {
		return
	}
	if resp.StatusCode != htt.StatusOK {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = fmt.Errorf(string(contents))
		return
	}
}
