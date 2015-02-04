package boxtools

import (
	"encoding/json"
	"fmt"
	"time"
)

type User struct {
	Id        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type Client struct {
	Id        int64
	User_id   int64
	Key       string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type Meta struct {
	Client_id int64
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
	User_id   int64
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

func ConvertMetaStructToFileStruct(metaStruct Meta) (fileStruct File, err error) {
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
