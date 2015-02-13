package structs

import "time"

type StateChange struct {
	File         File
	IsCreate     bool
	IsLocal      bool
	Quit         <-chan bool
	Done         chan<- bool
	Error        chan<- int
	PreviousHash string
}

type ChannelMessages struct {
	Quit  chan<- bool
	Done  <-chan bool
	Error <-chan int
}

type FileSystemState struct {
	FileActionId int64
	State        map[string]File
}

type User struct {
	Id             int64
	Email          string `sql:"type:text;"`
	HashedPassword string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      time.Time
}

type Client struct {
	Id                      int64
	UserId                  int64
	SessionKey              string
	Name                    string
	IsServer                bool
	LastSynchedFileActionId int64
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               time.Time
}

type FileAction struct {
	Id           int64
	ClientId     int64
	IsCreate     bool
	CreatedAt    time.Time
	PreviousHash string
	File         File
	FileId       int64
}

type File struct {
	Id        int64
	UserId    int64
	Name      string
	Hash      string
	Size      int64
	Modified  time.Time
	Path      string `sql:"type:text;"`
	CreatedAt time.Time
}

type FileSystemFile struct {
	Id     int64
	UserId int64
	FileId int64
	Path   string `sql:"type:text;"`
}
