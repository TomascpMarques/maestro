package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"

	// "github.com/golang-migrate/migrate/v4"
	// "github.com/mattn/go-sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func ConnectToDatabase(dbFilePath string) (db *sqlx.DB, err error, usable bool) {
	usable = true
	dbFilePath = filepath.Clean(dbFilePath)

	_, err = os.Stat(dbFilePath)
	if err != nil {
		slog.Info("database-setup", "creation", "file does not exist, creating")
		if _, err = os.Create(dbFilePath); err != nil {
			slog.Error("database-setup", "cause", "could not create database file, probably lack of permissions")
			usable = false
			return
		}
	}

	db, err = sqlx.Open("sqlite3", dbFilePath)

	// To allow the app to continue to function if the file usage fails,
	// we change to a memory storage tactic for the sqlite instance
	if err != nil {
		slog.Error("database-setup", "cause", "failed to use giver sqlite path")
		slog.Info("database-setup", "action", "changing into app memory stored sqlite")
		return sqlx.MustConnect("sqlite3", ":memory:"), err, true
	}

	return
}

func RunMigrations() (err error) {
	return
}
