/*
** Observer.go
** Author: Marin Alcaraz
** Mail   <marin.alcaraz@gmail.com>
** Started on  Mon Feb 09 14:36:00 2015 Marin Alcaraz
** Last update Thu Feb 12 14:16:31 2015 Marin Alcaraz
 */

package UDPush

import (
	"fmt"
	"net"
)

// Constants

// MAX NUMBER PER WATCHER ENGINE

const maxClients = 10

type update struct {
	status  bool
	ownerID int
}

// NotificationEngine interface for the notification system
// Defines the requirements to create a gobox
type NotificationEngine interface {
	Initialize(id string)
	Attach(Watcher) error
	Detach(Watcher) bool
	Notify()
}

// WatcherEngine Interface for watcher (Observer) system
// Defines the requirements to create a gobox
// notification watcher.
type WatcherEngine interface {
	Update()
}

//Pusher struct that satisfies the NotificationEngine interface
type Pusher struct {
	ServerID string
	BindedTo uint
	Watchers map[int]*Watcher
	Pending  bool
}

// Watcher Struct that satisfies the WatcherEngine
// This type requires an auth mecanism in order
// to work in a safe way
type Watcher struct {
	OwnerID    int
	ClientID   int
	SessionKey int
	Action     bool
}

// Methods for struct to satisfy the notificationEngine interface

//Initialize is a 'constructor' for the pusher struct
func (e *Pusher) Initialize(id string) {
	e.ServerID = id
	e.Watchers = make(map[int]*Watcher, maxClients)
}

//Attach Add a new Watcher to the notification slice
func (e *Pusher) Attach(w *Watcher) (err error) {
	//Check if Watchers is full
	if len(e.Watchers) == maxClients {
		return fmt.Errorf("[!] Error: Not enough space for new client")
	}
	//Check if element already exists
	if e.Watchers[w.ClientID] != nil {
		return fmt.Errorf("[!] Warning: client already monitored, skipping addition")
	}
	e.Watchers[w.ClientID] = w
	return nil
}

//Detach Remove a watcher from the notification slice
func (e *Pusher) Detach(w Watcher) (err error) {
	//Check if element already exists
	if e.Watchers[w.ClientID] != nil {
		e.Watchers[w.ClientID] = nil
		return nil
	}
	return fmt.Errorf("[!] Error: client doesn't exist")
}

//Notify Tell the watcher {clientID} to update
func (e *Pusher) Notify(owner int) {
	for _, k := range e.Watchers {
		//Is there a better way to do this? Dictionary and list inside?
		if k.OwnerID == owner {
			k.Action = true
			k.Update()
		}
	}
}

//Utilities for pusher

//ShowWatchers Print current watchers in pusher
func (e *Pusher) ShowWatchers() {
	fmt.Printf("Current watchers in %s:\n", e.ServerID)
	for _, k := range e.Watchers {
		fmt.Println("Watcher: ", k)
	}
}

// Methods for satisfiying the interface

// Update Get update from pusher... Golint forces me to do this
// http://tinyurl.com/lhzjvmm
func (w *Watcher) Update() {
	w.Action = true
}

//Network related methods

func getPendingUpdates() update {
	return update{status: true,
		ownerID: 1}
}

//HandleConnection keeps alive the UDP notification service between
//client and server
func handleConnection(conn net.Conn) error {
	for {
		//Check if there is something to update...
		//TODO: This function needs pairing
		out := getPendingUpdates()
		if out.status {
			//Write to client

			//Create an slice of bytes to contain the ownerID
			//Since we can only send []bytes we must to this
			notification := make([]byte, 1)
			notification[0] = byte(out.ownerID)

			//Send the notification and check the error
			_, err := conn.Write(notification)
			if err != nil {
				return fmt.Errorf("Error handleConnection: %s", err)
			}
		}
	}
	return nil
}

//InitUDPush 'constructs' the UDP notification engine
//The e on the reciever stands for event
func (e *Pusher) InitUDPush() error {
	connectionString := fmt.Sprintf("%s:%s", e.ServerID, e.BindedTo)
	ln, err := net.Listen("tcp", connectionString)
	if err != nil {
		return fmt.Errorf("Error at initUDPush: %s", err)
	}
	for {
		conn, err := ln.Accept()
		defer conn.Close()
		fmt.Println("Host connected: ", ln.Addr())
		if err != nil {
			return fmt.Errorf("Error at initUDPush: %s", err)
		}
		go handleConnection(conn)
	}
}
