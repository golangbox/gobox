package api

import (
	"fmt"
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
	fileActions, _ := boxtools.GenerateSliceOfRandomFileActions(1, 1, 10)

	var hashesToUpload []string
	var err error
	hashesToUpload, err = apiClient.SendFileActionsToServer(fileActions)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(hashesToUpload)
}
