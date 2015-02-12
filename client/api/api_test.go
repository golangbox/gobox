package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/golangbox/gobox/boxtools"
	server_api "github.com/golangbox/gobox/server/api"
	"github.com/golangbox/gobox/server/model"
	"github.com/golangbox/gobox/structs"
	"github.com/jinzhu/gorm"
)

var user structs.User
var client structs.Client
var apiClient Api

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

	apiClient = New(client.SessionKey)

	go server_api.ServeServerRoutes("8000")
}

func TestSendFileActionsToServer(t *testing.T) {
	fileActions, _ := boxtools.GenerateSliceOfRandomFileActions(1, 1, 1)

	var hashesToUpload []string
	var err error
	hashesToUpload, err = apiClient.SendFileActionsToServer(fileActions)
	if err != nil {
		t.Error(err)
	}

	if fileActions[0].File.Hash != hashesToUpload[0] {
		t.Error(fmt.Errorf("Wrong hash returned"))
	}
}

func TestDownloadFileFromServer(t *testing.T) {
	testFile := structs.File{
		Hash:   "fc45acaffc35a3aa674f7c0d5a03d22350b4f2ff4bf45ccebad077e5af80e512",
		UserId: user.Id,
	}
	model.DB.Create(&testFile)

	url, err := apiClient.DownloadFileFromServer("fc45acaffc35a3aa674f7c0d5a03d22350b4f2ff4bf45ccebad077e5af80e512")
	if err != nil {
		t.Error(err)
	}
	_ = url
	// futher tested to confirm functioanlity in
	// TestUploadFileToServer
}

func TestUploadFileToServer(t *testing.T) {
	file := []byte("this is a file")
	err := apiClient.UploadFileToServer(file)
	if err != nil {
		t.Error(err)
	}

	expectedHash := "fc45acaffc35a3aa674f7c0d5a03d22350b4f2ff4bf45ccebad077e5af80e512"
	testFile := structs.File{
		Hash:   expectedHash,
		UserId: user.Id,
	}
	model.DB.Create(&testFile)

	s3_url, err := apiClient.DownloadFileFromServer(
		expectedHash,
	)
	if err != nil {
		t.Error(err)
	}
	resp, err := http.Get(s3_url)
	if err != nil {
		t.Error(err)
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}

	if string(contents) != "this is a file" {
		t.Error(fmt.Errorf("S3 file contents don't match"))
	}
}
