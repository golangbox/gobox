/*
** Observer.go
** Author: Marin Alcaraz
** Mail   <marin.alcaraz@gmail.com>
** Started on  Mon Feb 09 14:36:00 2015 Marin Alcaraz
** Last update Mon Feb 09 17:27:39 2015 Marin Alcaraz
 */

package main

import "fmt"

// Constants

// MAX NUMBER PER WATCHER ENGINE

const maxClients = 5

// Interface for the notification system
// Defines the requirements to create a gobox
// notification engine

type notificationEngine interface {
	initialize(id string)
	attach(watcher) error
	detach(watcher) bool
	notify()
}

// Interface for watcher (Observer) system
// Defines the requirements to create a gobox
// notification watcher.

type watcherEngine interface {
	update()
}

//Struct that satisfies the NotificationEngine

type pusher struct {
	ServerID string
	Watchers map[int]*watcher
	status   int
}

//Struct that satisfies the WatcherEngine
// This type requires an auth mecanism in order
// to work in a safe way

type watcher struct {
	ownerID    int
	clientID   int
	sessionKey int
	action     bool
}

// Methods for struct to satisfy the notificationEngine interface

func (e *pusher) initialize(id string) {
	e.ServerID = id
	e.Watchers = make(map[int]*watcher, maxClients)
	e.status = 0
}

//Add a new Watcher to the notification slice
func (e *pusher) attach(w *watcher) (err error) {
	//Check if Watchers is full
	if len(e.Watchers) == maxClients {
		return fmt.Errorf("[!] Error: Not enough space for new client")
	}
	//Check if element already exists
	if e.Watchers[w.clientID] != nil {
		return fmt.Errorf("[!] Warning: client already monitored, skipping addition")
	}
	e.Watchers[w.clientID] = w
	return nil
}

//Remove a watcher from the notification slice
func (e *pusher) detach(w watcher) (err error) {
	//Check if element already exists
	if e.Watchers[w.clientID] != nil {
		e.Watchers[w.clientID] = nil
		return nil
	}
	return fmt.Errorf("[!] Error: client doesn't exist")
}

//Tell the watcher {clientID} to update
func (e *pusher) notify(owner int) {
	for _, k := range e.Watchers {
		//Is there a better way to do this? Dictionary and list inside?
		if k.ownerID == owner {
			k.action = true
			k.update()
		}
	}
}

//Utilities for pusher

//Print current watchers in pusher
func (e *pusher) showWatchers() {
	fmt.Printf("Current watchers in %s:\n", e.ServerID)
	for _, k := range e.Watchers {
		fmt.Println("Watcher: ", k)
	}
}

//Methods to satisfy the WatcherEngine interface

//Get update from pusher

func (w *watcher) update() {
	w.action = true
}

// Simulation

func main() {

	var n pusher
	var w1 watcher
	var w2 watcher
	var w3 watcher

	//Create a watcher
	w1.ownerID = 42
	w2.ownerID = 42
	w3.ownerID = 43

	w1.clientID = 1337
	w2.clientID = 1338
	w3.clientID = 1339

	//initialize the pusher
	n.initialize("Server 1")

	fmt.Println(n.attach(&w1))
	fmt.Println(n.attach(&w2))
	fmt.Println(n.attach(&w3))

	n.showWatchers()

	n.notify(42)
	n.detach(w2)

	//  w1 and w2 should be true
	n.showWatchers()

}
