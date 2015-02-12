package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/server/model"
	"github.com/golangbox/gobox/server/s3"
	"github.com/golangbox/gobox/structs"
	"github.com/jinzhu/gorm"
	"github.com/sqs/mux"
)

func ServeServerRoutes(port string) {
	r := mux.NewRouter()
	r.StrictSlash(true)

	// public
	r.HandleFunc("/", IndexHandler)
	r.HandleFunc("/login/", SignUpHandler).Methods("POST")
	r.HandleFunc("/sign-up/", LoginHandler).Methods("POST")

	// require client authentication
	r.HandleFunc("/file-actions/", sessionValidate(FileActionsHandler)).Methods("POST")
	r.HandleFunc("/upload/", sessionValidate(UploadHandler)).Methods("POST")
	r.HandleFunc("/download/", sessionValidate(FileDownloadHandler)).Methods("POST")
	r.HandleFunc("/clients/", sessionValidate(ClientsFileActionsHandler)).Methods("POST")

	// static files? (css, js, etc...)
	// r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	http.Handle("/", r)

	fmt.Println("Serving api on port :" + port)
	http.ListenAndServe(":"+port, nil)
}

type httpError struct {
	err            error
	code           int
	responseWriter http.ResponseWriter
}

func (h *httpError) check() bool {
	if h.err != nil {
		h.responseWriter.WriteHeader(h.code)
		h.responseWriter.Write([]byte(h.err.Error()))
		return true
	} else {
		return false
	}
}

func sessionValidate(fn func(http.ResponseWriter, *http.Request, structs.Client)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client, err := verifyAndReturnClient(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}
		fn(w, r, client)
	}
}

func verifyAndReturnClient(req *http.Request) (client structs.Client, err error) {
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

func FileActionsHandler(w http.ResponseWriter, req *http.Request,
	client structs.Client) {
	httpError := httpError{responseWriter: w}

	var contents []byte
	contents, httpError.err = ioutil.ReadAll(req.Body)
	httpError.code = http.StatusInternalServerError
	if httpError.check() {
		return
	}

	var fileActions []structs.FileAction
	httpError.err = json.Unmarshal(contents, &fileActions)
	httpError.code = http.StatusNotAcceptable
	if httpError.check() {
		return
	}

	for _, value := range fileActions {
		value.ClientId = client.Id
	}

	httpError.err = boxtools.WriteFileActionsToDatabase(fileActions, client)
	httpError.code = http.StatusInternalServerError
	if httpError.check() {
		return
	}

	var hashesThatNeedToBeUploaded []string
	hashMap := make(map[string]bool)
	// write to a map to remove any duplicate hashes
	for _, value := range fileActions {
		hashMap[value.File.Hash] = true
	}
	for key, _ := range hashMap {
		var exists bool
		exists, httpError.err = s3.TestKeyExistence(key)
		httpError.code = http.StatusInternalServerError
		if httpError.check() {
			return
		}
		if exists == false {
			hashesThatNeedToBeUploaded = append(
				hashesThatNeedToBeUploaded,
				key,
			)
		}
	}

	var jsonBytes []byte
	jsonBytes, httpError.err = json.Marshal(hashesThatNeedToBeUploaded)
	httpError.code = http.StatusInternalServerError
	if httpError.check() {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
func UploadHandler(w http.ResponseWriter, req *http.Request,
	client structs.Client) {
	httpError := httpError{responseWriter: w}

	var contents []byte
	contents, httpError.err = ioutil.ReadAll(req.Body)
	httpError.code = http.StatusUnauthorized
	if httpError.check() {
		return
	}

	h := sha256.New()
	_, httpError.err = h.Write(contents)
	httpError.code = http.StatusInternalServerError
	if httpError.check() {
		return
	}
	byteString := h.Sum(nil)
	sha256String := hex.EncodeToString(byteString)

	// we have the hash, so we might as well check if it
	// exists again before we upload
	var exists bool
	exists, httpError.err = s3.TestKeyExistence(sha256String)
	if httpError.check() {
		return
	}
	if exists == false {
		httpError.err = s3.UploadFile(sha256String, contents)
		if httpError.check() {
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func FileDownloadHandler(w http.ResponseWriter, req *http.Request,
	client structs.Client) {
	httpError := httpError{responseWriter: w}
	httpError.code = http.StatusInternalServerError

	var user structs.User
	query := model.DB.Model(&client).Related(&user)
	httpError.err = query.Error
	if httpError.check() {
		return
	}

	fileHash := req.FormValue("fileHash")
	if fileHash == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	var file structs.File
	query = model.DB.Where(
		&structs.File{
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

	var exists bool
	exists, httpError.err = s3.TestKeyExistence(fileHash)
	if httpError.check() {
		return
	}

	if exists != true {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var url string
	url, httpError.err = s3.GenerateSignedUrl(fileHash)
	if httpError.check() {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(url))
}

func ClientsFileActionsHandler(w http.ResponseWriter, req *http.Request,
	client structs.Client) {
	httpError := httpError{responseWriter: w}
	httpError.code = http.StatusInternalServerError

	var user structs.User
	query := model.DB.Model(&client).Related(&user)
	httpError.err = query.Error
	if httpError.check() {
		return
	}

	var clients []structs.Client
	query = model.DB.Model(&user).Not("Id = ?", client.Id).Related(&clients, "clients")
	httpError.err = query.Error
	if httpError.check() {
		return
	}

	var clientIds []int64
	for _, value := range clients {
		clientIds = append(clientIds, value.Id)
	}

	var fileActions []structs.FileAction
	query = model.DB.Where("client_id in (?)", clientIds).
		Where("Id > ?", client.LastSynchedFileActionId).
		Find(&fileActions)
	httpError.err = query.Error
	if httpError.check() {
		return
	}

	var responseJsonBytes []byte
	responseJsonBytes, httpError.err = json.Marshal(fileActions)
	if httpError.check() {
		return
	}
	w.Write(responseJsonBytes)

	var highestId int64
	for _, value := range fileActions {
		if value.Id > highestId {
			highestId = value.Id
		}
	}
	client.LastSynchedFileActionId = highestId
	model.DB.Save(&client)

}

func IndexHandler(w http.ResponseWriter, req *http.Request) {

}

func SignUpHandler(w http.ResponseWriter, req *http.Request) {

}

func LoginHandler(w http.ResponseWriter, req *http.Request) {

}
