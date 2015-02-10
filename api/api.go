package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/model"
	"github.com/golangbox/gobox/s3"
	"github.com/jinzhu/gorm"
	"github.com/sqs/mux"
)

func ServeServerRoutes(port string) {
	r := mux.NewRouter()
	// "http://thing.com/login" redirects to "http://thing.com/login/"
	r.StrictSlash(true)

	// public
	r.HandleFunc("/", IndexHandler)
	r.HandleFunc("/login/", SignUpHandler).Methods("POST")
	r.HandleFunc("/login/", LoginHandler).Methods("POST")

	// require client authentication
	r.HandleFunc("/file-actions/", FileActionsHandler).Methods("POST")
	r.HandleFunc("/upload/", UploadHandler).Methods("POST")
	r.HandleFunc("/download/", FileDownloadHandler).Methods("POST")
	r.HandleFunc("/clients/", ClientsFileActionsHandler).Methods("POST")

	// static files? (css, js, etc...)
	// r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	http.Handle("/", r)

	fmt.Println("Serving api on port :" + port)
	http.ListenAndServe(":"+port, nil)
}

func verifyAndReturnClient(req *http.Request) (client model.Client, err error) {
	sessionKey := req.FormValue("sessionKey")
	if sessionKey == "" {
		err = fmt.Errorf("No session key with request")
		return
	}
	query := model.DB.Where("session_key = ?", sessionKey).First(&client)
	if query.Error != nil {
		err = query.Error
		return
	}
	if client.Id == 0 {
		err = fmt.Errorf("No client matching this session key")
		return
	}
	return
}

func FileActionsHandler(w http.ResponseWriter, req *http.Request) {
	client, err := verifyAndReturnClient(req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	contents, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var fileActions []model.FileAction
	err = json.Unmarshal(contents, &fileActions)
	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte(err.Error()))
		return
	}

	for _, value := range fileActions {
		value.ClientId = client.Id
	}

	err = boxtools.WriteFileActionsToDatabase(fileActions, client)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var hashesThatNeedToBeUploaded []string

	hashMap := make(map[string]bool)
	// write to a map to remove any duplicate hashes
	for _, value := range fileActions {
		hashMap[value.File.Hash] = true
	}
	for key, _ := range hashMap {
		exists, err := s3.TestKeyExistence(key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		if exists == false {
			hashesThatNeedToBeUploaded = append(
				hashesThatNeedToBeUploaded,
				key,
			)
		}
	}

	w.WriteHeader(http.StatusOK)
	jsonBytes, err := json.Marshal(hashesThatNeedToBeUploaded)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(jsonBytes)

	// req.Body
}
func UploadHandler(w http.ResponseWriter, req *http.Request) {
	_, err := verifyAndReturnClient(req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	contents, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	h := sha256.New()
	_, err = h.Write(contents)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	byteString := h.Sum(nil)

	sha256String := hex.EncodeToString(byteString)

	// we have the hash, so we might as well check if it
	// exists again before we upload
	exists, err := s3.TestKeyExistence(sha256String)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	if exists == false {
		err = s3.UploadFile(sha256String, contents)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func FileDownloadHandler(w http.ResponseWriter, req *http.Request) {
	client, err := verifyAndReturnClient(req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}

	var user model.User
	query := model.DB.Model(&client).Related(&user)
	if query.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(query.Error.Error()))
		return
	}

	fileHash := req.FormValue("fileHash")
	fmt.Println(fileHash)
	if fileHash == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	var file model.File
	query = model.DB.Where(
		&model.File{
			UserId: user.Id,
			Hash:   fileHash,
		},
	).First(&file)
	if query.Error != nil {
		if query.Error == gorm.RecordNotFound {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(query.Error.Error()))
		}
		return
	}
	url, err := s3.GenerateSignedUrl(fileHash)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(url))
}

func ClientsFileActionsHandler(w http.ResponseWriter, req *http.Request) {
	client, err := verifyAndReturnClient(req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	_ = client

}

func IndexHandler(w http.ResponseWriter, req *http.Request) {

}

func SignUpHandler(w http.ResponseWriter, req *http.Request) {

}

func LoginHandler(w http.ResponseWriter, req *http.Request) {

}
