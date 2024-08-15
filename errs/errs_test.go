package errs

import (
	"errors"
	"testing"
)

type CompressionErrorVariant uint

const (
	FailedCreatingRootZipFile CompressionErrorVariant = iota
	FailedCreatingBackupZipFile
	FailedCopyingFilesIntoArchive
)

func (m CompressionErrorVariant) Error() string {
	switch m {
	case FailedCopyingFilesIntoArchive:
		return "failed copying files into archive"
	case FailedCreatingBackupZipFile:
		return "failed creating backup zip file"
	case FailedCreatingRootZipFile:
		return "failed creating root zip file"
	}
	return "Unknown Error"
}

type CompressionError struct {
	CustomError
}

func NewCompressionError(variant CompressionErrorVariant, cause, message string) *CompressionError {
	return &CompressionError{
		NewCustomError(variant, cause, message),
	}
}

func TestSomeCustomError(t *testing.T) {
	someCachedError := NewCompressionError(FailedCopyingFilesIntoArchive, "Unexpected", "Something wen't wrong")
	someOtherCachedError := NewCompressionError(FailedCreatingBackupZipFile, "Unexpected", "Something wen't wrong")

	someErroringFunc := func() error {
		return NewCompressionError(FailedCopyingFilesIntoArchive, "Unexpected", "Something wen't wrong")
	}

	err := someErroringFunc()

	if err == nil {
		t.Error("Failed to return an error")
	}

	if !errors.Is(err, FailedCopyingFilesIntoArchive) {
		t.Error("Failed errors.Is implementation")
	}

	if !errors.Is(err, someCachedError) {
		t.Error("Failed basic errors.Is implementation")
	}

	if errors.Is(err, someOtherCachedError) {
		t.Error("Failed basic errors.Is implementation")
	}

}
