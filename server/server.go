package server

import (
	"fmt"
	"log"

	"github.com/golangbox/gobox/UDPush"
	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/server/api"
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
}

func (s *server) checkStatus() bool {
	if s.services.api == true &&
		s.services.s3 == true &&
		s.services.udpush == true {
		return true
	}
	return false
}

func createDummyUser() error {
	user, err := boxtools.NewUser("gobox@gmail.com", "password")
	if err != nil {
		return err
	}
	_ = user
	return nil
}

//Run creates all the structures to make or project work
func Run() {

	//Launch API

	s := server{
		name:        "Elvis",
		ip:          "127.0.0.1",
		port:        4242,
		clientLimit: 10,
	}

	// model.DB, _ = gorm.Open(
	// 	"postgres",
	// 	"dbname=gobox sslmode=disable",
	// )
	// model.DB.AutoMigrate(
	// 	&structs.User{},
	// 	&structs.Client{},
	// 	&structs.FileAction{},
	// 	&structs.File{},
	// 	&structs.FileSystemFile{},
	// )

	err := createDummyUser()
	if err != nil {
		log.Fatal(err)
	}
	////Launch UDP notification service
	////Define the Subject (The guy who is goin to hold all the clients)

	pusher := &UDPush.Pusher{
		ServerID: s.ip,
		BindedTo: s.port,
	}

	go func() {
		err = pusher.InitUDPush()
		if err != nil {
			fmt.Println(err)
		}
	}()
	api.ServeServerRoutes("8000", pusher)
}
