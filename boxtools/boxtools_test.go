package boxtools

import (
	"fmt"
	"github.com/golangbox/gobox/model"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"math/rand"
	"path/filepath"
	"testing"
	"time"
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

	client, err := NewClient(user, "test", false)

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

func RandomString(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += string('a' + rand.Intn(26))
	}
	return s
}

func GenerateFilePathFromRoot(root string) string {
	depth := 3
	s := root
	for i := 0; i < depth; i++ {

		s += string(filepath.Separator)
		s += RandomString(rand.Intn(10) + 1)
	}
	return s
}

func GenerateRandomFile(user_id int) (file model.File, err error) {
	path := GenerateFilePathFromRoot("/path")
	basename := filepath.Base(path)
	h, err := GenerateRandomSha256()
	if err != nil {
		return file, err
	}
	return model.File{
		UserId:    int64(user_id),
		Name:      basename,
		Hash:      h,
		Size:      rand.Int63(),
		Modified:  time.Now(),
		Path:      path,
		CreatedAt: time.Now(),
	}, err

}

func GenerateRandomFileAction(client_id int, user_id int, isCreate bool) (fileAction model.FileAction, err error) {
	file, err := GenerateRandomFile(user_id)
	if err != nil {
		return fileAction, err
	}
	return model.FileAction{
		ClientId:  int64(client_id),
		IsCreate:  isCreate,
		CreatedAt: time.Now(),
		File:      file,
	}, err
}

func GenerateSliceOfRandomFileActions(user_id int, clients int, actions int) (fileActions []model.FileAction, err error) {
	fileActions = make([]model.FileAction, actions)
	for i := 0; i < int(actions); i++ {
		isAction := rand.Intn(2) == 1
		action, err := GenerateRandomFileAction(rand.Intn(clients)+1, user_id, isAction)
		if err != nil {
			return fileActions, err
		}
		fileActions[i] = action
	}
	return fileActions, err
}

func GenerateNoisyAndNonNoisyFileActions(user_id int, clients int, totalNonNoisyActions int, createPairs int) (nonNoisyActions []model.FileAction,
	noisyActions []model.FileAction, err error) {
	numNoisyActions := totalNonNoisyActions + createPairs
	nonNoisyActions, err = GenerateSliceOfRandomFileActions(user_id, clients, totalNonNoisyActions)
	if err != nil {
		return
	}
	noisyActions = make([]model.FileAction, numNoisyActions)
	copy(noisyActions, nonNoisyActions)
	offset := len(nonNoisyActions)
	for i := 0; i < createPairs; i++ {
		new := nonNoisyActions[i]
		new.IsCreate = !new.IsCreate
		noisyActions[i+offset] = new
	}
	return

}

func TestRemoveRedundancy(t *testing.T) {
	_, noisy, err := GenerateNoisyAndNonNoisyFileActions(1, 4, 10, 10)
	if err != nil {
		t.Log("Could not generate file actions successfully")
		t.FailNow()
	}
	result := RemoveRedundancyFromFileActions(noisy)
	if 0 != len(result) {
		t.Log("Result should be empty")
		t.FailNow()
	}
	_, noisy, err = GenerateNoisyAndNonNoisyFileActions(1, 4, 10, 5)
	result = RemoveRedundancyFromFileActions(noisy)
	if 5 != len(result) {
		t.Log("Result of cleaning should be length 5")
		t.FailNow()
	}
}
