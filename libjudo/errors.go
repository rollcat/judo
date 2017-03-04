package libjudo

type TimeoutError struct {
}
type CancelError struct {
}

func (e TimeoutError) Error() string {
	return "Operation timed out"
}

func (e CancelError) Error() string {
	return "Operation cancelled"
}
