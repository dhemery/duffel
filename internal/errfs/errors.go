package errfs

import (
	"errors"
	"fmt"
)

// Errors to return from corresponding FS and File methods.
var (
	ErrLstat    = generalOpErr(lstatOp)    // General error for Lstat.
	ErrOpen     = generalOpErr(openOp)     // General error for Open.
	ErrReadDir  = generalOpErr(readDirOp)  // General error for ReadDir.
	ErrReadLink = generalOpErr(readLinkOp) // General error for ReadLink.
	ErrWrite    = generalOpErr(writeOp)    // General error for directory write oprations.
	ErrStat     = generalOpErr(statOp)     // General error for Stat.
	errGeneral  = errors.New("general error")
)

// Error represents an error associated with an errfs operation. To
// associate an Error with a file, pass the Error when creating the
// file or adding it to the FS. Subsequent calls to the operation for
// that file will return the Error, whether the call is initiated on
// the file itself or on the FS.
type Error struct {
	op  string
	err error
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.op, e.err)
}

func (e Error) Unwrap() error {
	return e.err
}

// LstatErr wraps err in an Error for Lstat.
func LstatErr(err error) Error {
	return Error{lstatOp, err}
}

// OpenErr wraps err in an Error for Open.
func OpenErr(err error) Error {
	return Error{openOp, err}
}

// ReadDirErr wraps err in an Error for ReadDir.
func ReadDirErr(err error) Error {
	return Error{readDirOp, err}
}

// ReadLinkErr wraps err in an Error for ReadLink.
func ReadLinkErr(err error) Error {
	return Error{readLinkOp, err}
}

// RemoveErr wraps err in an Error for Remove.
func RemoveErr(err error) Error {
	return Error{readLinkOp, err}
}

// StatErr wraps err in an Error for Stat.
func StatErr(err error) Error {
	return Error{statOp, err}
}

func generalOpErr(op string) Error {
	return Error{op, errGeneral}
}
