package watcher

import (
	"errors"

	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-fsnotify/fsnotify"
	"github.com/golangbox/gobox/structs"
)

const (
	CREATE = 0
	MODIFY = 1
	DELETE = 2
)

type RecursiveWatcher struct {
	*fsnotify.Watcher
	Files   chan structs.StateChange
	Folders chan string
}

func NewRecursiveWatcher(path string) (*RecursiveWatcher, error) {
	folders := Subfolders(path)
	if len(folders) == 0 {
		return nil, errors.New("No folders to watch.")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	rw := &RecursiveWatcher{Watcher: watcher}

	rw.Files = make(chan structs.StateChange, 10)
	rw.Folders = make(chan string, len(folders))

	for _, folder := range folders {
		rw.AddFolder(folder)
	}
	return rw, nil
}

func (watcher *RecursiveWatcher) AddFolder(folder string) {
	err := watcher.Add(folder)
	if err != nil {
		log.Println("Error watching: ", folder, err)
	}
	watcher.Folders <- folder
}

func createLocalStateChange(path string, eventType int) (change structs.StateChange, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		if !(eventType == DELETE && os.IsNotExist(err)) {
			return
		}
	}
	change.IsCreate = (eventType != DELETE)
	change.IsLocal = true
	change.File.Path = path
	if eventType != DELETE {
		change.File.Name = fi.Name()
		change.File.Size = fi.Size()
		change.File.Modified = fi.ModTime()
		// hmmm, what do we do if the file wasn't created? os.Stat doesn't provide created
		if eventType == CREATE {
			change.File.CreatedAt = fi.ModTime()
		}
	}
	return
}

func (watcher *RecursiveWatcher) Run(debug bool) {
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if ext := filepath.Ext(event.Name); ext == ".tmp" {
					continue
				}
				absPath, err := filepath.Abs(event.Name)
				if err == nil {
					event.Name = absPath
				}

				// create a file/directory
				if event.Op&fsnotify.Create == fsnotify.Create {
					fi, err := os.Stat(event.Name)
					if err != nil {
						// eg. stat .subl513.tmp : no such file or directory
						if debug {
							// DebugError(err)
						}
					} else if fi.IsDir() {
						if debug {
							// DebugMessage("Detected new directory %s", event.Name)
						}
						if !shouldIgnoreFile(filepath.Base(event.Name)) {
							watcher.AddFolder(event.Name)
						}
					} else {
						if debug {
							// DebugMessage("Detected new file %s", event.Name)
						}

						change, err := createLocalStateChange(event.Name, CREATE)
						if err != nil {
							continue
						}
						watcher.Files <- change
					}
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					// modified a file, assuming that you don't modify folders
					if debug {
						// DebugMessage("Detected file modification %s", event.Name)
					}

					change, err := createLocalStateChange(event.Name, MODIFY)
					if err != nil {
						continue
					}
					watcher.Files <- change
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					change, err := createLocalStateChange(event.Name, DELETE)
					if err != nil {
						continue
					}
					watcher.Files <- change

				}

			case err := <-watcher.Errors:
				log.Println("error", err)
			}
		}
	}()
}

// Subfolders returns a slice of subfolders (recursive), including the folder provided.
func Subfolders(path string) (paths []string) {
	filepath.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			// skip folders that begin with a dot
			if shouldIgnoreFile(name) && name != "." && name != ".." {
				return filepath.SkipDir
			}
			paths = append(paths, newPath)
		}
		return nil
	})
	return paths
}

// shouldIgnoreFile determines if a file should be ignored.
// File names that begin with "." or "_" are ignored by the go tool.
func shouldIgnoreFile(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
