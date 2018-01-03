package main

import (
	"errors"
)

// ErrorTimeout Operation has timed out
var ErrorTimeout = errors.New("Operation timed out")

// ErrorCancel Operation was canceled while pending
var ErrorCancel = errors.New("Operation canceled")
