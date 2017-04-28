package libjudo

import (
	"errors"
)

var TimeoutError = errors.New("Operation timed out")
var CancelError = errors.New("Operation cancelled")
