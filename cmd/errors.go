package cmd

// ErrExpiredCreds represents an error when the saved credentials have been expired.
type ErrExpiredCreds struct {
	Msg string
}

// Satisfy the error interface.
func (e ErrExpiredCreds) Error() string {
	return e.Msg
}

// newErrExpiredCreds creates a new ErrExpiredCreds with the appropriate error.
func newErrExpiredCreds() ErrExpiredCreds {
	return ErrExpiredCreds{Msg: "Expired credentials. Please signin again."}
}
