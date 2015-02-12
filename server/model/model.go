package model

import (
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

var DB gorm.DB

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
