package s3

import (
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	validS3Key        = "test"
	validS3KeyContent = "asdfasdfasdfasldkjfhalskdjhfalsjkdhflaksjdhflasjkdfha\ndsfa\nsdfhlasjdhflaskjdhflasjhdflkjh"
)

func TestTestKeyExistence(t *testing.T) {
	var exists bool
	var err error

	exists, err = TestKeyExistence(validS3Key)
	if exists != true {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
	exists, err = TestKeyExistence("notvalid")
	if exists == true {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateSignedUrl(t *testing.T) {

	url, err := GenerateSignedUrl(validS3Key)
	_ = url
	if err != nil {
		t.Error(err)
	}

	resp, err := http.Get(url)
	contents, err := ioutil.ReadAll(resp.Body)
	if validS3KeyContent != string(contents) {
		t.Fail()
	}
}

func TestFileUpload(t *testing.T) {
	testString := "file thing"
	file := []byte(testString)
	err := UploadFile("test2", file)
	if err != nil {
		t.Error(err)
	}
	getByte, err := bucket.Get("test2")
	if err != nil {
		t.Error(err)
	}
	if string(file) != string(getByte) {
		t.Fail()
	}
}
