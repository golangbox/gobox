package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golangbox/gobox/structs"
)

const (
	goBoxDirectory     = "."
	goBoxDataDirectory = ".GoBox"
	serverEndpoint     = "http://requestb.in/1mv9fa41"
	// serverEndpoint           = "http://www.google.com"
	filesystemCheckFrequency = 5
)

func main() {
	fmt.Println("Running GoBox...")

	createGoBoxLocalDirectory()

	go monitorFiles()

	select {}

}

func createGoBoxLocalDirectory() {

	_, err := os.Stat(goBoxDataDirectory)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Making directory")
		err := os.Mkdir(goBoxDataDirectory, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func monitorFiles() {

	var newfileInfos = make(map[string]structs.File)
	var fileInfos = make(map[string]structs.File)

	data, err := ioutil.ReadFile(goBoxDataDirectory + "/data")
	if err != nil {
		fmt.Println(err)
	}
	if data != nil {
		err = json.Unmarshal(data, &fileInfos)
		if err != nil {
			fmt.Println(err)
		}
	}
	for _ = range time.Tick(filesystemCheckFrequency * time.Second) {
		fmt.Println("Checking to see if there are any filesystem changes.")
		newfileInfos, err = findFilesInDirectory(goBoxDirectory)
		if err != nil {
			fmt.Println(err)
		}
		err := compareFileInfos(fileInfos, newfileInfos)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Error uploading file changes, skipping this upload cycle")
		} else {
			err = writefileInfosToLocalFile(newfileInfos)
			if err != nil {
				fmt.Println(err)
			} else {
				// fmt.Println("Saved data locally")
			}
			fileInfos = newfileInfos
		}
	}
}

func writefileInfosToLocalFile(fileInfos map[string]structs.File) error {
	jsonBytes, err := json.Marshal(fileInfos)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(goBoxDataDirectory+"/data", jsonBytes, 0644)
	return err
}

func handleFileChange(isCreate bool, file structs.File) (err error) {
	infoToSend := structs.FileAction{
		IsCreate: isCreate,
		File:     file,
	}
	resp, err := uploadMetadata(infoToSend)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error uploading metadata, status code: %d", resp.StatusCode)
		return
	}

	return
}

func uploadMetadata(uploadinfo structs.FileAction) (resp *http.Response, err error) {
	fmt.Println("Uploading metadata: (" + uploadinfo.File.Name + ")")
	jsonBytes, err := json.Marshal(uploadinfo)
	if err != nil {
		return
	}
	resp, err = http.Post(serverEndpoint+"/meta", "application/json", bytes.NewBuffer(jsonBytes))
	fmt.Println(resp)
	if err != nil {
		return resp, err
	}
	if uploadinfo.IsCreate == true {
		var contents []byte
		contents, err = ioutil.ReadAll(resp.Body)
		err := resp.Body.Close()
		if err != nil {
			return resp, err
		}
		fmt.Println(string(contents))
		if string(contents) == "true" {
			resp, err = uploadFile(uploadinfo.File.Path)
			if err != nil {
				return resp, err
			}
		} else {
			fmt.Println("no need for upload")
		}
	}
	if err != nil {
		return
	}
	return
}

func compareFileInfos(fileInfos map[string]structs.File,
	newfileInfos map[string]structs.File) (err error) {

	for key, value := range newfileInfos {
		// http://stackoverflow.com/questions/2050391/how-to-test-key-existence-in-a-map
		if _, exists := fileInfos[key]; !exists {
			fmt.Println("Need to Upload:", value.Name)
			err = handleFileChange(true, value)
			if err != nil {
				return err
			}
		}
	}
	for key, value := range fileInfos {
		if _, exists := newfileInfos[key]; !exists {
			fmt.Println("Need to delete:", key)
			err = handleFileChange(false, value)
			if err != nil {
				return err
			}
		}
	}
	return
}

// http://matt.aimonetti.net/posts/2013/07/01/golang-multipart-file-upload-example/
func newfileUploadRequest(uri string, params map[string]string,
	paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	return req, err
}

func getSha256FromFilename(filename string) (sha256String string,
	err error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Error reading file for sha256: %s", err)
	}
	h := sha256.New()
	_, err = h.Write(file)
	if err != nil {
		return "", fmt.Errorf("Error writing file to hash for sha256: %s", err)
	}
	byteString := h.Sum(nil)

	sha256String = hex.EncodeToString(byteString)

	return sha256String, nil
}

func uploadFile(name string) (*http.Response, error) {
	// filename := "main.go"
	file, _ := os.Stat(name)
	s, err := getSha256FromFilename(name)
	if err != nil {
		fmt.Println(err)
	}

	size := strconv.Itoa(int(file.Size()))
	extraParams := map[string]string{
		"Name": file.Name(),
		"Hash": s,
		"Size": size,
	}

	// http://requestb.in/19w82ne1
	//"http://10.0.7.205:4242/upload"
	request, err := newfileUploadRequest("http://10.0.7.205:4242/upload", extraParams, "FileName", name)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	// resp, err := client.Do(request)
	return client.Do(request)
}

func mapKeyValue(path string, sha256 string) (key string) {
	return path + "-" + sha256
}

func findFilesInDirectoryHelper(directory string,
	fileInfos map[string]structs.File) (outputfileInfos map[string]structs.File, err error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("Unable to read directory: %s", err)
	}

	for _, f := range files {
		name := f.Name()
		path := directory + "/" + name

		// fmt.Println("Scanning file: " + path)
		if f.IsDir() {
			if f.Name() != goBoxDataDirectory {
				fileInfos, err = findFilesInDirectoryHelper(path, fileInfos)
			}
		} else {
			sha256, err := getSha256FromFilename(path)
			if err != nil {
				fmt.Println(err)
			}

			fileInfos[mapKeyValue(path, sha256)] = structs.File{
				Name:     name,
				Hash:     sha256,
				Size:     f.Size(),
				Path:     path,
				Modified: f.ModTime(),
			}
		}
	}

	return fileInfos, err
}

func findFilesInDirectory(directory string) (outputfileInfos map[string]structs.File, err error) {
	emptyfileInfos := make(map[string]structs.File)
	return findFilesInDirectoryHelper(directory, emptyfileInfos)
}
