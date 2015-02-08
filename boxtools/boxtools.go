package boxtools

import (
	"encoding/json"
	"fmt"
	// "reflect"
	"time"

	"github.com/jinzhu/gorm"

	"code.google.com/p/go.crypto/bcrypt"
)

type User struct {
	Id             int64
	Email          string `sql:"type:text;"`
	HashedPassword string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Client struct {
	ID        int64
	UserID    int64
	Key       string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type JournalEntry struct {
	ID     int64
	Task   string
	FileID int64
}

type CurrentFile struct {
	ID     int64
	UserID int64
	FileID int64
}

type Meta struct {
	ClientID  int64
	Task      string
	Name      string
	Hash      string
	Size      int64
	Path      string `sql:"type:text;"`
	Modified  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type File struct {
	ID        int64
	Name      string
	Hash      string
	Size      int64
	Path      string `sql:"type:text;"`
	Modified  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FileInfo struct {
	Name     string
	Hash     string
	Size     int64
	Path     string
	Modified time.Time
}

type UploadInfo struct {
	Task string
	File FileInfo
}

func ConvertMetaStructToFileStruct(metaStruct Meta) (fileStruct File, err error) {
	return File{
		Name:      metaStruct.Name,
		Hash:      metaStruct.Hash,
		Size:      metaStruct.Size,
		Path:      metaStruct.Path,
		Modified:  metaStruct.Modified,
		CreatedAt: metaStruct.CreatedAt,
		UpdatedAt: metaStruct.UpdatedAt,
	}, err
}

func DBCreateFileFromMetaStruct(metaStruct Meta, db gorm.DB) (File, error) {
	file, err := ConvertMetaStructToFileStruct(metaStruct)
	if err != nil {
		return file, err
	}

	query := db.FirstOrCreate(&file)
	if query.Error != nil {
		return file, query.Error
	}
	return file, err
}

func DBCreateJournalEntry(task string, fileID int64, db gorm.DB) (JournalEntry, error) {
	journalStruct := JournalEntry{
		Task:   task,
		FileID: fileID,
	}
	query := db.Create(journalStruct)
	return journalStruct, query.Error
}

func CreateJournalEntryFromMeta(metaStruct Meta, db gorm.DB) (JournalEntry, error) {
	file, err := DBCreateFileFromMetaStruct(metaStruct, db)
	if err != nil {
		return JournalEntry{}, err
	}
	return DBCreateJournalEntry(metaStruct.Task, file.ID, db)
}

func DBFindFile(metaStruct Meta, db gorm.DB) (file File, err error) {
	query := db.Where("hash = ?", metaStruct.Hash).First(&file)

	if query.Error != nil {
		return file, query.Error
	}
	return file, err
}

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

func ConvertJsonStringToMetaStruct(jsonMeta string) (metadata Meta, err error) {
	data := []byte(jsonMeta)
	var unmarshalledUploadInfo UploadInfo
	err = json.Unmarshal(data, &unmarshalledUploadInfo)
	if err != nil {
		err = fmt.Errorf("Error unmarshaling json to fileInfo: %s", err)
		return metadata, err
	}
	return Meta{
		1,
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
		if exists == true {
			metaMap[opposingMapKey] = value - 1
		} else {
			simplifiedMetadata = append(simplifiedMetadata, meta)
		}
	}
	return
}

func ComputeFilesFromMetadata(metadata []Meta) (filedata []File) {
	return
}
