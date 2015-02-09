package model

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

var DB gorm.DB

type User struct {
	Id             int64
	Email          string `sql:"type:text;"`
	HashedPassword string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      time.Time
}

type Client struct {
	Id         int64
	User_id    int64
	SessionKey string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  time.Time
}

type FileAction struct {
	Id        int64
	Client_id int64
	IsCreate  bool
	File_id   string
	CreatedAt time.Time
}

type File struct {
	Id        int64
	User_id   int64
	Name      string
	Hash      string
	Size      int64
	Path      string `sql:"type:text;"`
	CreatedAt time.Time
}

type FileSystemFile struct {
	Id      int64
	User_id int64
	File_id int64
	Path    string `sql:"type:text;"`
}

func main() {
	db, err := gorm.Open("postgres", "dbname=gobox sslmode=disable")
	if err != nil {
		fmt.Println(err)
	}
	query := db.AutoMigrate(&User{}, &Client{})
	if query.Error != nil {
		fmt.Println(query.Error)
	}
	fmt.Println(query)

	// meta, err := convertJsonStringToMetaStruct(testJsonString)
	// fmt.Println(meta)
	// file, err := convertMetaStructToFileStruct(meta)
	// fmt.Println(file)
}
