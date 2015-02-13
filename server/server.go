package main

import (
	"fmt"

	"github.com/golangbox/gobox/UDPush"
	"github.com/golangbox/gobox/server/api"
	"github.com/golangbox/gobox/server/model"
	"github.com/golangbox/gobox/structs"
	"github.com/jinzhu/gorm"
)

type services struct {
	s3     bool
	api    bool
	udpush bool
}

type server struct {
	services
	name        string
	ip          string
	port        uint
	clientLimit uint
	status      func() bool
	display     func() string
	pusher      UDPush.Pusher
}

func (s *server) checkStatus() bool {
	if s.services.api == true &&
		s.services.s3 == true &&
		s.services.udpush == true {
		return true
	}
	return false
}

func main() {

	//Launch API
	//api.ServeServerRoutes()

	s := server{
		name:        "Elvis",
		ip:          "127.0.0.1",
		port:        4242,
		clientLimit: 10,
	}

	////Launch UDP notification service
	////Define the Subject (The guy who is goin to hold all the clients)

	s.pusher = UDPush.Pusher{
		ServerID: s.ip,
		BindedTo: s.port,
	}

	err := s.pusher.InitUDPush()
	if err != nil {
		fmt.Println(err)
	}

	model.DB, _ = gorm.Open(
		"postgres",
		"dbname=gobox sslmode=disable",
	)
	model.DB.AutoMigrate(
		&structs.User{},
		&structs.Client{},
		&structs.FileAction{},
		&structs.File{},
		&structs.FileSystemFile{},
	)

	api.ServeServerRoutes("8000")
}
