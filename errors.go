package htracker

import (
	"errors"
)

var ErrNotExist = errors.New("the item could not be found")
var ErrAlreadyExists = errors.New("the item already exists")
