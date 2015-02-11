package main

import (
	// "bytes"
	// "crypto/sha256"
	// "encoding/hex"
	// "encoding/json"
	// "errors"
	"fmt"
	// "io"
	// "io/ioutil"
	// "log"
	// "mime/multipart"
	// "net/http"
	// "os"
	// "path/filepath"
	// "strconv"
	// "strings"
	// "time"

	// "github.com/go-fsnotify/fsnotify"
	"github.com/golangbox/gobox/client/watcher"
	// "github.com/golangbox/gobox/model"
)

// once the client starts an upload with the server, that is the point of no return, if the file gets changed
// the server has to hash and sort that out

// therefore, if fsnotify triggers an event on a file that is uploading, wait until the file is done sending
// and then send it

// use a

const (
	goBoxDirectory     = "."
	goBoxDataDirectory = ".GoBox"
	serverEndpoint     = "http://requestb.in/1mv9fa41"
	// serverEndpoint           = "http://www.google.com"
	filesystemCheckFrequency = 5
)

func main() {
	fmt.Printf("%+v\n", watcher.RecursiveWatcher{})
}
