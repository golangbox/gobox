package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/golangbox/gobox/UDPush"
	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/server/model"
	"github.com/golangbox/gobox/server/s3"
	"github.com/golangbox/gobox/structs"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

var T *template.Template

func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	T.ExecuteTemplate(w, tmpl+".html", data)
}

var Pusher *UDPush.Pusher

func ServeServerRoutes(port string, pusher *UDPush.Pusher) {
	Pusher = pusher
	var err error
	T, err = template.ParseGlob("server/templates/*")
	_ = err
	r := mux.NewRouter()
	r.StrictSlash(true)

	// public
	r.HandleFunc("/", IndexHandler)
	r.HandleFunc("/login/", LoginHandler).Methods("GET")
	r.HandleFunc("/sign-up/", SignUpHandler).Methods("POST")
	r.HandleFunc("/file-data/{email}", FilesHandler).Methods("POST")
	r.HandleFunc("/download/{id}/{filename}", DownloadHandler).Methods("GET")

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
		log.Fatal(h.err)
		return true
	} else {
		return false
	}
}

func sessionValidate(fn func(http.ResponseWriter, *http.Request, structs.Client)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
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
	sessionKey := req.FormValue("SessionKey")
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

	fileActions, httpError.err = boxtools.WriteFileActionsToDatabase(fileActions, client)
	httpError.code = http.StatusInternalServerError
	if httpError.check() {
		return
	}

	var user structs.User
	query := model.DB.Model(&client).Related(&user)
	httpError.err = query.Error
	if httpError.check() {
		return
	}

	Pusher.Notify(client.SessionKey)

	errs := boxtools.ApplyFileActionsToFileSystemFileTable(fileActions, user)
	if len(errs) != 0 {
		fmt.Println(errs)
	}

	var hashesThatNeedToBeUploaded []string
	hashMap := make(map[string]bool)
	// write to a map to remove any duplicate hashes
	for _, value := range fileActions {
		if value.IsCreate == true {
			hashMap[value.File.Hash] = true
		}
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

	lastIdString := req.FormValue("lastId")
	if lastIdString == "" {
		httpError.err = fmt.Errorf("Need last fileAction Id.")
		if httpError.check() {
			return
		}
	}

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
		Where("Id > ?", lastIdString).
		Find(&fileActions)
	httpError.err = query.Error
	if httpError.check() {
		return
	}

	var highestId int64
	for _, value := range fileActions {
		if value.Id > highestId {
			highestId = value.Id
		}
	}

	fileActions = boxtools.RemoveRedundancyFromFileActions(fileActions)

	for key, value := range fileActions {
		var file structs.File
		_ = model.DB.First(&file, value.FileId)
		fileActions[key].File = file
	}

	responseStruct := structs.ClientFileActionsResponse{
		LastId:      highestId,
		FileActions: fileActions,
	}

	var responseJsonBytes []byte
	responseJsonBytes, httpError.err = json.Marshal(responseStruct)
	if httpError.check() {
		return
	}
	w.Write(responseJsonBytes)

	model.DB.Save(&client)

}

func FilesHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	email := vars["email"]

	var user structs.User
	query := model.DB.Where("email = ?", email).First(&user)

	var files []structs.FileSystemFile
	query = model.DB.Where("user_id = ?", user.Id).Find(&files)

	for i, value := range files {
		query = model.DB.First(&files[i].File, value.FileId)
	}

	if query.Error != nil {
		panic(query.Error)
	}
	jsonBytes, _ := json.Marshal(files)
	w.Write(jsonBytes)
}

func DownloadHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	intId, _ := strconv.Atoi(id)
	int64Id := int64(intId)

	var file structs.File
	query := model.DB.First(&file, int64Id)
	_ = query
	url, err := s3.GenerateSignedUrl(file.Hash)
	resp, err := http.Get(url)
	// keyBytes, err := ioutil.ReadAll(resp.Body)
	w.Header().Add("Content-Type", "application/octet-stream")
	io.Copy(w, resp.Body)
	_ = err
	// w.Write(keyBytes)

	// w.Write()
	// fmt.Println(query.Error)
}

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	RenderTemplate(w, "index", nil)
}

func SignUpHandler(w http.ResponseWriter, req *http.Request) {
	// wants username and pass1 and pass2 posted as a form?
	// returns 200 or  some sort of error to client?

}

func LoginHandler(w http.ResponseWriter, req *http.Request) {
	httpError := httpError{responseWriter: w}
	httpError.code = http.StatusInternalServerError

	var user structs.User
	model.DB.First(&user)
	client, err := boxtools.NewClient(user, "stephen", false)
	httpError.err = err
	if httpError.check() {
		return
	}

	w.Write([]byte(client.SessionKey))
}
