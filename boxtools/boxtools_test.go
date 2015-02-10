package boxtools

import (
	"fmt"
	"testing"

	"github.com/golangbox/gobox/model"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

const (
	email    = "max.t.mcdonnell@gmail.com"
	password = "password"
)

var user model.User
var client model.Client

func init() {
	var err error
	model.DB, err = gorm.Open("postgres", "dbname=goboxtest sslmode=disable")

	model.DB.DropTableIfExists(&model.User{})
	model.DB.DropTableIfExists(&model.Client{})
	model.DB.DropTableIfExists(&model.FileAction{})
	model.DB.DropTableIfExists(&model.File{})
	model.DB.DropTableIfExists(&model.FileSystemFile{})
	model.DB.AutoMigrate(&model.User{}, &model.Client{}, &model.FileAction{}, &model.File{}, &model.FileSystemFile{})

	if err != nil {
		fmt.Println(err)
	}
}

func TestUserCreation(t *testing.T) {
	var err error
	user, err = NewUser(email, password)
	if err != nil {
		t.Error(err)
	}
	if user.Email != email {
		t.Fail()
	}
	if user.HashedPassword == "" {
		t.Fail()
	}
}

func TestClientCreation(t *testing.T) {
	var user model.User
	model.DB.Where("email = ?", email).Find(&user)

	client, err := NewClient(user)

	if err != nil {
		t.Error(err)
	}
	user = model.User{} //nil user

	//testing relation
	model.DB.Model(&client).Related(&user)
	if user.Email != email {
		t.Fail()
	}
}

func TestPasswordValidation(t *testing.T) {
	user, err := ValidateUserPassword(email, password)
	if err != nil {
		t.Error(err)
	}
	if user.Email != email {
		t.Fail()
	}
	// clean up created user
	model.DB.Where("email = ?", email).Delete(model.User{})
}

func TestJsonMetaConversion(t *testing.T) {
}

func TestRemoveRedundancy(t *testing.T) {
}
