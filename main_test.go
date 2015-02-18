package main

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/golangbox/gobox/server"
	"github.com/golangbox/gobox/server/model"
	"github.com/golangbox/gobox/structs"
	"github.com/jinzhu/gorm"
)

func init() {
	model.DB, _ = gorm.Open("postgres", "dbname=goboxtest sslmode=disable")

	model.DB.DropTableIfExists(&structs.User{})
	model.DB.DropTableIfExists(&structs.Client{})
	model.DB.DropTableIfExists(&structs.FileAction{})
	model.DB.DropTableIfExists(&structs.File{})
	model.DB.DropTableIfExists(&structs.FileSystemFile{})
	model.DB.AutoMigrate(
		&structs.User{},
		&structs.Client{},
		&structs.FileAction{},
		&structs.File{},
		&structs.FileSystemFile{},
	)

}
func TestEverything(t *testing.T) {
	go server.Run()
	time.Sleep(time.Second * 2)
	paths := []string{
		"sandbox/client1/",
		"sandbox/client2/",
	}
	for _, value := range paths {
		go func(value string) {
			cmd := exec.Command(
				"go",
				"run",
				"client/client.go",
				value)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}(value)
	}
	select {}
}
