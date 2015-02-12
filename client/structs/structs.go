package structs

import (
	"github.com/golangbox/gobox/model"
)

type StateChange struct {
	File     model.File
	begin    <-chan bool
	quit     <-chan bool
	done     <-chan bool
	IsCreate bool
	IsLocal  bool
}

type ActionsOnPath struct {
	local  StateChange
	remote StateChange
}
