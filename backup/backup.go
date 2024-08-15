package backup

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/TomascpMarques/maestro/errs"
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
	errs.CustomError
}

func NewCompressionError(variant CompressionErrorVariant, cause, message string) *CompressionError {
	return &CompressionError{
		errs.NewCustomError(variant, cause, message),
	}
}

func compressFile(file *os.File) *CompressionError {
	// https://pkg.go.dev/github.com/kdungs/zip
	zipDestFile, err := os.Create(fmt.Sprintf("%s.zip", file.Name()))
	if err != nil {
		slog.Error("backup-file-compress", "file-creation-fail", "failed to create the destination zip file")
		return NewCompressionError(FailedCreatingRootZipFile, "zip creation failure", "failed to create the destination zip file")
	}
	defer zipDestFile.Close()

	zipWriter := zip.NewWriter(zipDestFile)
	defer zipWriter.Close()

	zipW, err := zipWriter.Create(fmt.Sprintf("%s.backup", filepath.Base(file.Name())))
	if err != nil {
		slog.Error("backup-file-compress", "zip-creation-fail", "failed to create a zip file for the backup")
		return NewCompressionError(FailedCreatingBackupZipFile, "zip backup failure", "failed to create backup zip file")
	}

	if _, err = io.Copy(zipW, file); err != nil {
		slog.Error("backup-file-compress", "zip-write-fail", "failed to write to the zip file")
		return NewCompressionError(FailedCopyingFilesIntoArchive, "copying into zip", "failed copy backup zip into archive")
	}

	return nil
}

type BackupLocations struct {
	SourceLocation string
	BackupLocation string
}

func backupFile(locations BackupLocations) (err error) {
	fileName := filepath.Base(locations.SourceLocation)
	destinationFilename := filepath.Clean(
		fmt.Sprintf(
			"%s/%s-bkup",
			locations.BackupLocation,
			filepath.Base(locations.SourceLocation),
		),
	)
	destinationBkpFileName := filepath.Base(destinationFilename)

	slog.Info("backup-file", "init-backup", fmt.Sprintf("starting backing up file [%s] into [%s]", fileName, destinationFilename))

	err = os.MkdirAll(locations.BackupLocation, 0740)
	if err != nil {
		slog.Error("backup-file", "create-destination", "failed to create back-up destination")
		return
	}

	original, err := os.Open(locations.SourceLocation)
	if err != nil {
		slog.Error("backup-file", "read-backup-operation", "failed to read the file to be backed-up")
		return
	}
	defer original.Close()

	destinationBkpFile, err := os.Create(destinationFilename)
	if err != nil {
		slog.Error("backup-file", "write-backup-operation", "failed to create the destination file for the backup")
		return
	}
	defer destinationBkpFile.Close()

	const BUFFER_SIZE = 5324288 // 5 Mebibyte
	READ_BUFFER := make([]byte, BUFFER_SIZE)
	bufferedReader := bufio.NewReader(original)

	for {
		readN, readErr := bufferedReader.Read(READ_BUFFER)
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			slog.Error("backup-file", "read-backup-operation", "failed to read a part of the file into a buffer")
			err = readErr
			break
		}
		if readN == 0 {
			break
		}

		_, err = destinationBkpFile.Write(READ_BUFFER)
		if err != nil {
			slog.Error("backup-file", "write-backup-operation", "failed to write the buffer contents into the file")
			return
		}
	}

	slog.Info("backup-file", "finished-backup", fmt.Sprintf("done backing up file [%s], success", fileName))

	if compressionError := compressFile(destinationBkpFile); compressionError != nil {
		if errors.Is(compressionError, FailedCreatingRootZipFile) {
			slog.Warn(
				"backup-file-compress", "compress-fail",
				fmt.Sprintf("compressing file [%s], failed, but backup exists", destinationBkpFileName),
			)
			return
		}
	}

	slog.Info("backup-file", "finished-backup-compression", "Successfully compressed the backup file")

	if os.Remove(destinationBkpFileName) != nil {
		slog.Warn("backup-file", "failed-deleting-temp-file", "")
	}

	return
}

/* Backup task status codes and task emitted signals */
type TaskStatus uint

const (
	BackupSuccess TaskStatus = 1 + iota
	BackupFailed
	BackupEnded
	BackupPaused
	BackupSkipped
)

type BackupTaskSignal struct {
	Done   bool
	Status TaskStatus
	Error  error
}

// ---------------------------------------------------

/* Task handling signals, sent from the requester of the task */

type TaskHandleSignal uint

const (
	EndBackupTask TaskHandleSignal = 1 + iota
	PauseBackupTask
	SkipBackupTask
)

// ----------------------------------------------------

func CreateFileBackupTask(backups BackupLocations, taskHandle <-chan TaskHandleSignal, backupInterval time.Duration) /* Returns */ (
	signalTheHandler chan<- BackupTaskSignal,
	ticker *time.Ticker,
) {
	ticker = time.NewTicker(backupInterval)
	go func() {
		skipBackup := false
		pauseBackup := false
		for {
			select {
			case taskSignal := <-taskHandle:
				switch taskSignal {
				case EndBackupTask:
					signalTheHandler <- BackupTaskSignal{
						Done:   true,
						Status: BackupEnded,
						Error:  nil,
					}
					ticker.Stop()
					slog.Info(
						"database-backup",
						"behaviour-termination",
						fmt.Sprintf("terminating all backups, requested at %s", time.UTC.String()),
					)
					return
				case PauseBackupTask:
					signalTheHandler <- BackupTaskSignal{
						Done:   false,
						Status: BackupPaused,
						Error:  nil,
					}
					pauseBackup = true
					slog.Info(
						"database-backup",
						"behaviour-change",
						fmt.Sprintf("pausing all following backups, requested at %s", time.UTC.String()),
					)
					continue
				case SkipBackupTask:
					signalTheHandler <- BackupTaskSignal{
						Done:   false,
						Status: BackupSkipped,
						Error:  nil,
					}
					slog.Info(
						"database-backup",
						"behaviour-termination",
						fmt.Sprintf("skipping all following backups, requested at %s", time.UTC.String()),
					)
					skipBackup = true
				}

			case <-ticker.C:
				if skipBackup {
					slog.Info(
						"database-backup",
						"behaviour-change",
						fmt.Sprintf("skipped backup at %s", time.UTC.String()),
					)
					skipBackup = false
					continue
				}
				if pauseBackup {
					slog.Info(
						"database-backup",
						"behaviour-change",
						fmt.Sprintf("backup is paused, skipped backup at %s", time.UTC.String()),
					)
					continue
				}

				err := backupFile(backups)
				if err != nil {
					signalTheHandler <- BackupTaskSignal{
						Done:   false,
						Status: BackupFailed,
						Error:  err,
					}
					continue
				}
				// After backup is done and successful, warn any observer
				signalTheHandler <- BackupTaskSignal{
					Done:   false,
					Status: BackupSuccess,
					Error:  nil,
				}
			}
		}
	}()
	return
}
