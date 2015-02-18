package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"text/template"

	"github.com/golangbox/gobox/UDPush"
	"github.com/golangbox/gobox/boxtools"
	"github.com/golangbox/gobox/server/model"
	"github.com/golangbox/gobox/server/s3"
	"github.com/golangbox/gobox/structs"
	"github.com/jinzhu/gorm"
	"github.com/sqs/mux"
)

var T *template.Template

func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	T.ExecuteTemplate(w, tmpl+".html", data)
}

var Pusher UDPush.Pusher

func ServeServerRoutes(port string, pusher *UDPush.Pusher) {
	fmt.Println("SERVE ROUTES")
	pusher.ShowWatchers()
	var err error
	T, err = template.ParseGlob("templates/*")
	_ = err
	r := mux.NewRouter()
	r.StrictSlash(true)

	// public
	r.HandleFunc("/", IndexHandler)
	r.HandleFunc("/login/", LoginHandler).Methods("GET")
	r.HandleFunc("/sign-up/", SignUpHandler).Methods("POST")
	r.HandleFunc("/file-data/", FilesHandler).Methods("POST")

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

	var clients []structs.Client
	model.DB.Where("user_id = ?", user.Id).
		Not("id = ?", client.Id).
		Find(&clients)
	//How to get the right clients to notify?
	for _, value := range clients {
		Pusher.Notify(value.SessionKey)
	}

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
	var client structs.Client
	query := model.DB.Where("name =  ?", "test").First(&client)
	var user structs.User
	query = model.DB.Where("id = ?", client.UserId).First(&user)
	fmt.Println(user)
	var files structs.FileSystemFile
	query = model.DB.Where("user_id = ?", user.Id).Find(&files)
	if query.Error != nil {
		panic(query.Error)
	}
	jsonBytes, _ := json.Marshal(files)
	w.Write(jsonBytes)
}

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	RenderTemplate(w, "index", nil)
}

func SignUpHandler(w http.ResponseWriter, req *http.Request) {
	// wants username and pass1 and pass2 posted as a form?
	// returns 200 or  some sort of error to client?

}

func LoginHandler(w http.ResponseWriter, req *http.Request) {
	var client structs.Client
	model.DB.Where("name = ?", "test").First(&client)
	w.Write([]byte(client.SessionKey))
}
