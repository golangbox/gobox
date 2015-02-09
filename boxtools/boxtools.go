package boxtools

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/jinzhu/gorm"

	"crypto/sha1"

	"code.google.com/p/go.crypto/bcrypt"
)

func NewUser(email string, password string, db gorm.DB) (user User, err error) {
	hash, err := hashPassword(password)
	if err != nil {
		return
	}
	user = User{
		Email:          email,
		HashedPassword: hash,
	}
	query := db.Create(&user)
	if query.Error != nil {
		return user, query.Error
	}
	return
}

func NewClient(user User, db gorm.DB) (client Client, err error) {
	// calculate key if we need a key?
	client = Client{
		User_id: user.id,
	}
	io.W
}

func NewClientKey() (key string) {
	h := sha1.New()
	// rand.Seed(time.Now().Unix())
	// rand.P
	// io.WriteString(h, rand.Float64().String())
	io.WriteString(w, s)
	thing := h.Sum(nil)
	return string(thing)
}

func ValidateUserPassword(email, password string, db gorm.DB) (user User, err error) {
	db.Where("email = ?", email).First(&user)
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

func ConvertJsonStringToMetaStruct(jsonMeta string, client Client) (metadata Meta, err error) {
	data := []byte(jsonMeta)
	var unmarshalledUploadInfo UploadInfo
	err = json.Unmarshal(data, &unmarshalledUploadInfo)
	if err != nil {
		err = fmt.Errorf("Error unmarshaling json to fileInfo: %s", err)
		return metadata, err
	}
	return Meta{
		client.Id,
		unmarshalledUploadInfo.Task,
		unmarshalledUploadInfo.File.Name,
		unmarshalledUploadInfo.File.Hash,
		unmarshalledUploadInfo.File.Size,
		unmarshalledUploadInfo.File.Path,
		unmarshalledUploadInfo.File.Modified,
		time.Now(),
		time.Now(),
	}, err
}

func ConvertMetaStructToJsonString(metaStruct Meta) (metaJson string, err error) {
	uploadInfoStruct := UploadInfo{
		metaStruct.Task,
		FileInfo{
			metaStruct.Name,
			metaStruct.Hash,
			metaStruct.Size,
			metaStruct.Path,
			metaStruct.Modified,
		},
	}
	jsonBytes, err := json.Marshal(uploadInfoStruct)
	if err != nil {
		return
	}
	metaJson = string(jsonBytes)
	return
}

func ConvertMetaStructToFileStruct(metaStruct Meta, db gorm.DB) (fileStruct File, err error) {
	var clientid int
	if metaStruct.Client_id != 0 {

	}
	return File{
		metaStruct.Client_id, //WRONG
		metaStruct.Name,
		metaStruct.Hash,
		metaStruct.Size,
		metaStruct.Path,
		metaStruct.Modified,
		time.Now(),
		time.Now(),
	}, err
}

func RemoveRedundancyFromMetadata(metadata []Meta) (simplifiedMetadata []Meta) {
	// removing redundancy
	// if a file is created, and then deleted remove

	var metaMap = make(map[string]int)

	for i, meta := range metadata {
		// create a map of the metadata
		// key values are task+path+hash
		// hash map value is the number of occurences
		_, _ = i, meta
		mapKey := meta.Task + meta.Path + meta.Hash
		value, exists := metaMap[mapKey]
		if exists {
			metaMap[mapKey] = value + 1
		} else {
			metaMap[mapKey] = 1
		}
	}

	for i, meta := range metadata {
		// for each meta value, check if a pair exists in the map
		// if it does remove one iteration of that value
		// if it doesn't write that to the simplified array

		// This ends up removing pairs twice, once for each matching value
		_, _ = i, meta
		var opposingTask string
		if meta.Task == "delete" {
			opposingTask = "upload"
		} else {
			opposingTask = "delete"
		}
		opposingMapKey := opposingTask + meta.Path + meta.Hash
		value, exists := metaMap[opposingMapKey]
		if exists == true && value > 0 {
			metaMap[opposingMapKey] = value - 1
		} else {
			simplifiedMetadata = append(simplifiedMetadata, meta)
		}
	}
	return
}

func ComputeFilesFromMetadata(metadata []Meta) (filedata []File) {
	simplifiedMetadata := RemoveRedundancyFromMetadata(metadata)
	for _, value := range metadata {
		filedata = append(filedata, convertMetaStructToFileStruct(value))
	}
	return
}

func ApplyMetadataToFilesTable(metadata []Meta, user User, db gorm.DB) (err error) {
	for _, meta := range metadata {
		var file File
		if meta.Task == "delete" {
			query := db.Where("path = ?", meta.Path).First(&file)
			if query.Error != nil {
				// error
			}
			if file.Hash != meta.Hash {
				fmt.Println("uh oh")
				// this should never happen.
				// What's up with your filesystem?
			}
			db.Delete(&file)
		} else if meta.Task == "upload" {
			file, err = ConvertMetaStructToFileStruct(meta)
			db.Create(&file)
		}
	}
}
