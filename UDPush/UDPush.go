/*
** Observer.go
** Author: Marin Alcaraz
** Mail   <marin.alcaraz@gmail.com>
** Started on  Mon Feb 09 14:36:00 2015 Marin Alcaraz
** Last update Thu Feb 19 15:45:08 2015 Marin Alcaraz
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
	Watchers map[string]Watcher
	Pending  bool
}

// Watcher Struct that satisfies the WatcherEngine
// This type requires an auth mecanism in order
// to work in a safe way
type Watcher struct {
	OwnerID    int
	ClientID   int
	SessionKey string
	Action     bool
	Connection net.Conn
}

// Methods for struct to satisfy the notificationEngine interface

//Initialize is a 'constructor' for the pusher struct
func (e *Pusher) Initialize(id string) {
	e.ServerID = id
	e.Watchers = make(map[string]Watcher, maxClients)
}

//Attach Add a new Watcher to the notification slice
func (e *Pusher) Attach(w Watcher) (err error) {
	//Check if Watchers is full
	if len(e.Watchers) == maxClients {
		return fmt.Errorf("[!] Error: Not enough space for new client")
	}
	//Check if element already exists
	if _, k := e.Watchers[w.SessionKey]; k {
		return fmt.Errorf("[!] Warning: client already monitored, skipping addition")
	}
	fmt.Println("Client registered ", w.SessionKey)
	e.Watchers[w.SessionKey] = w
	return nil
}

//Detach Remove a watcher from the notification slice
func (e *Pusher) Detach(w Watcher) (err error) {
	//Check if element already exists
	if item, ok := e.Watchers[w.SessionKey]; ok {
		item.Connection.Close()
		delete(e.Watchers, w.SessionKey)
		return nil
	}
	return fmt.Errorf("[!] Error: client doesn't exist")
}

//Notify Tell the watcher {clientID} to update
func (e *Pusher) Notify(sessionkey string) {
	for _, k := range e.Watchers {
		if k.SessionKey != sessionkey {
			k.Update()
			k.Action = true
		}
	}
}

//Utilities for pusher

//ShowWatchers Print current watchers in pusher
func (e *Pusher) ShowWatchers() {
	for _, k := range e.Watchers {
		fmt.Println("Watcher: ", k)
	}
}

// Methods for satisfiying the interface

// Update Get update from pusher... Golint forces me to do this
// http://tinyurl.com/lhzjvmm
func (w *Watcher) Update() {
	w.Action = true
	fits := w.SessionKey[:2]
	fmt.Println("[!] Attempting to update watcher: %s", fits)
	writen, err := w.Connection.Write([]byte("Y"))
	if writen != len([]byte("Y")) {
		fmt.Println("[!]Error writting: unable to write")
	}
	if err != nil {
		fmt.Printf("%s", err)
	}

}

//Network related methods

func getPendingUpdates() update {
	return update{status: true,
		ownerID: 1}
}

//InitUDPush 'constructs' the UDP notification engine
//The e on the reciever stands for event
func (e *Pusher) InitUDPush() error {
	//Initialize the map
	e.Watchers = make(map[string]Watcher, maxClients)
	connectionString := fmt.Sprintf("%s:%d", e.ServerID, e.BindedTo)
	ln, err := net.Listen("tcp", connectionString)
	if err != nil {
		return fmt.Errorf("Error at initUDPush: %s", err)
	}
	fmt.Println("[+] UDP Listening on: ", connectionString)
	for {
		conn, err := ln.Accept()
		fmt.Println("Host connected: ", ln.Addr())
		if err != nil {
			return fmt.Errorf("Error at initUDPush: %s", err)
		}
		session := make([]byte, 64)
		conn.Read(session)
		e.Attach(Watcher{
			SessionKey: string(session),
			Connection: conn,
		})
	}
}
