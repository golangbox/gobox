package boxtools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/golangbox/gobox/model"

	"code.google.com/p/go.crypto/bcrypt"
)

var salt []byte

func init() {
	f, err := os.Open("/dev/random")
	if err != nil {
		panic("Unable to open /dev/random")
	}
	salt = make([]byte, 8)
	n, err := f.Read(salt)
	if n != 8 || err != nil {
		panic("Couldn't read from dev/random")
	}
	f.Close()

	rand.Seed(time.Now().Unix())

}

func NewUser(email string, password string) (user model.User, err error) {
	hash, err := hashPassword(password)
	if err != nil {
		return
	}
	user = model.User{
		Email:          email,
		HashedPassword: hash,
	}
	query := model.DB.Create(&user)
	if query.Error != nil {
		return user, query.Error
	}
	client, err := NewClient(user, "Server", true)
	if err != nil {
		return
	}
	return
}

func NewClient(user model.User, name string, isServer bool) (client model.Client, err error) {
	// calculate key if we need a key?
	newKey, err := GenerateRandomSha256()
	if err != nil {
		return
	}
	client = model.Client{
		UserId:     user.Id,
		SessionKey: newKey,
		IsServer:   isServer,
		Name:       name,
	}
	query := model.DB.Create(&client)
	if query.Error != nil {
		return client, query.Error
	}
	return
}

func GenerateRandomSha256() (s string, err error) {
	h := sha256.New()
	h.Write(salt)
	io.WriteString(h, time.Now().String())
	bytes := h.Sum(nil)
	s = hex.EncodeToString(bytes)
	return s, err
}

func ValidateUserPassword(email, password string) (user model.User, err error) {
	model.DB.Where("email = ?", email).First(&user)
	bytePassword := []byte(password)
	byteHash := []byte(user.HashedPassword)
	err = bcrypt.CompareHashAndPassword(byteHash, bytePassword)
	return user, err
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}

func hashPassword(password string) (hash string, err error) {
	bytePassword := []byte(password)
	defer clear(bytePassword)
	byteHash, err := bcrypt.GenerateFromPassword(bytePassword, bcrypt.DefaultCost)
	return string(byteHash), err
}

func ConvertJsonStringToFileActionsStruct(jsonFileAction string, client model.Client) (fileAction model.FileAction, err error) {
	// {"Id":0,"ClientId":0,"IsCreate":true,"CreatedAt":"0001-01-01T00:00:00Z","File":{"Id":0,"UserId":0,"Name":"client.go","Hash":"f953d35b6d8067bf2bd9c46017c554b73aa28a549fac06ba747d673b2da5bfe0","Size":6622,"Modified":"2015-02-09T14:39:22-05:00","Path":"./client.go","CreatedAt":"0001-01-01T00:00:00Z"}}
	data := []byte(jsonFileAction)
	var unmarshalledFileAction model.FileAction
	err = json.Unmarshal(data, &unmarshalledFileAction)
	if err != nil {
		err = fmt.Errorf("Error unmarshaling json to model.FileAction: %s", err)
		return
	}
	return
}

func ConvertFileActionStructToJsonString(fileActionStruct model.FileAction) (fileActionJson string, err error) {
	jsonBytes, err := json.Marshal(fileActionStruct)
	if err != nil {
		return
	}
	fileActionJson = string(jsonBytes)
	return
}

func RemoveRedundancyFromFileActions(fileActions []model.FileAction) (simplifiedFileActions []model.FileAction) {
	// removing redundancy
	// if a file is created, and then deleted remove

	var actionMap = make(map[string]int)

	var isCreateMap = make(map[bool]string)
	isCreateMap[true] = "1"
	isCreateMap[false] = "0"

	for _, action := range fileActions {
		// create a map of the fileaction
		// key values are task+path+hash
		// hash map value is the number of occurences
		actionKey := isCreateMap[action.IsCreate] + action.File.Path + action.File.Hash
		value, exists := actionMap[actionKey]
		if exists {
			actionMap[actionKey] = value + 1
		} else {
			actionMap[actionKey] = 1
		}
	}

	for _, action := range fileActions {
		// for each action value, check if a pair exists in the map
		// if it does remove one iteration of that value
		// if it doesn't write that to the simplified array

		// This ends up removing pairs twice, once for each matching value
		var opposingTask string
		if action.IsCreate == true {
			opposingTask = isCreateMap[false]
		} else {
			opposingTask = isCreateMap[true]
		}
		opposingMapKey := opposingTask + action.File.Path + action.File.Hash
		value, exists := actionMap[opposingMapKey]
		if exists == true && value > 0 {
			actionMap[opposingMapKey] = value - 1
		} else {
			simplifiedFileActions = append(simplifiedFileActions, action)
		}
	}
	return
}

func ComputeFilesFromFileActions(fileActions []model.FileAction) (files []model.File) {
	simplifiedFileActions := RemoveRedundancyFromFileActions(fileActions)
	for _, value := range simplifiedFileActions {
		files = append(files, value.File)
	}
	return
}

func WriteFileActionsToDatabase(fileActions []mode.FileAction) (err error) {
	for _, fileAction := range fileActions {
		model.DB.Create(&fileAction)
		// need to look up in the file database and see if there's matching "File's"
		// to reference
	}
}

func ApplyFileActionsToFilesTable(fileActions []model.FileAction, user model.User) (err error) {
	// this is all wrong,
	// you're looking up the file in the file database
	// hmmm
	for _, fileAction := range fileActions {
		if fileAction.IsCreate == true {
			// what if the path is the same?
			model.DB.Create(&fileAction.File)
		} else {
			var file model.File
			query := model.DB.Where("path = ?", fileAction.File.Path).First(&file)
			if query.Error != nil {
				// uh oh
			}
			if file.Hash != fileAction.File.Hash {
				// uh oh
			}
			model.DB.Delete(&file)
		}
	}
	return
}
