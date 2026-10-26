package main

import (
	"errors"
)

// ErrorTimeout Operation has timed out
var ErrorTimeout = errors.New("operation timed out")

// ErrorCancel Operation was canceled while pending
var ErrorCancel = errors.New("operation canceled")
