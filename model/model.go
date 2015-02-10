package model

import (
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
	UserId     int64
	SessionKey string
	Name       string
	IsServer   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  time.Time
}

type FileAction struct {
	Id        int64
	ClientId  int64
	IsCreate  bool
	CreatedAt time.Time
	File      File
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
	Id      int64
	User_id int64
	File_id int64
	Path    string `sql:"type:text;"`
}

func main() {
	// var err error
	// DB, err = gorm.Open("postgres", "dbname=gobox sslmode=disable")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// query := DB.AutoMigrate(&User{}, &Client{})
	// if query.Error != nil {
	// 	fmt.Println(query.Error)
	// }
	// fmt.Println(query)

	// meta, err := convertJsonStringToMetaStruct(testJsonString)
	// fmt.Println(meta)
	// file, err := convertMetaStructToFileStruct(meta)
	// fmt.Println(file)
}
