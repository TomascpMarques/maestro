package backup

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type BackupLocations struct {
	SourceLocation string
	BackupLocation string
}

type FileBackupError uint

func backupFile(locations BackupLocations) (err error) {
	fileName := filepath.Base(locations.SourceLocation)
	destinationFilename := filepath.Clean(
		fmt.Sprintf(
			"%s/%s-bkup",
			locations.BackupLocation,
			filepath.Base(locations.SourceLocation),
		),
	)
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

	destination, err := os.Create(destinationFilename)
	if err != nil {
		slog.Error("backup-file", "write-backup-operation", "failed to create the destination file for the backup")
		return
	}
	defer destination.Close()

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

		_, err = destination.Write(READ_BUFFER)
		if err != nil {
			slog.Error("backup-file", "write-backup-operation", "failed to write the buffer contents into the file")
			return
		}
	}

	slog.Info("backup-file", "finished-backup", fmt.Sprintf("done backing up file [%s], success", fileName))
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
	signal chan<- BackupTaskSignal,
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
					signal <- BackupTaskSignal{
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
					signal <- BackupTaskSignal{
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
					signal <- BackupTaskSignal{
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
					signal <- BackupTaskSignal{
						Done:   false,
						Status: BackupSuccess,
						Error:  nil,
					}
					continue
				}
				// After backup is done and successful, warn any observer
				signal <- BackupTaskSignal{
					Done:   false,
					Status: BackupSuccess,
					Error:  nil,
				}
			}
		}
	}()
	return
}
