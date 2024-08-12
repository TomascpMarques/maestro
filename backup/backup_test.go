package backup

import (
	"log"
	"os"
	"path/filepath"

	"testing"

	"github.com/stretchr/testify/assert"
)

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func TestBackUpFile(t *testing.T) {
	basePath := "./.testing-backup/"

	// Create temp testing source folder
	sourcePath := filepath.Join(basePath, "source/")
	err := os.MkdirAll(sourcePath, 0744)
	handleErr(err)
	log.Printf("Created origin folder: %s", sourcePath)

	// Create temp testing destination folder
	destPath := filepath.Join(basePath, "dest/")
	err = os.MkdirAll(destPath, 0744)
	handleErr(err)
	log.Printf("Created destination folder: %s", destPath)

	sourceFilePath := filepath.Join(sourcePath, "source_sql")
	_, err = os.Create(sourceFilePath)
	handleErr(err)

	err = backupFile(BackupLocations{
		SourceLocation: sourceFilePath,
		BackupLocation: destPath,
	})

	if err != nil {
		t.Errorf("Failed to backup file: %e", err)
	}

	destDir, err := os.ReadDir(destPath)
	handleErr(err)
	for _, dir := range destDir {
		if !dir.IsDir() {
			assert.Equal(t, dir.Name(), "source_sql-bkup")
		}
	}

	// cleanUp
	_ = os.RemoveAll(basePath)
}
