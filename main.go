package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	backup "github.com/TomascpMarques/maestro/backup"
	gin "github.com/gin-gonic/gin"
	validator "github.com/go-playground/validator/v10"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
)

/*
	TODO Add a way to publish the current date-and-time to the raspberry pi, so we can keep track of it with some certainty
*/

// Struct validation across the entire app
var VALIDATE *validator.Validate = validator.New(
	validator.WithRequiredStructEnabled(),
	validator.WithPrivateFieldValidation(),
)

func main() {
	// Env file config loading
	configPath, defined := os.LookupEnv("ENV_PATH")
	if !defined {
		log.Fatalf("No Environment file specified")
	}
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Config Error:\n%s\n", err.Error())
	}
	// Will be used to later store the config values used to start this app instance
	configJson, err := json.Marshal(config)
	if err != nil {
		panic("Should not fail to parse config to json!")
	}

	// Prepare the basic telemetry/logging for the app
	telemetryFilePath, telemetryFile, err := TelemetryWriterFromFilePath(config.TelemetryConfig.Destination)
	if err != nil {
		log.Println(err)
		panic("Should not fail to create a writer for the telemetry file!")
	}
	defer telemetryFile.Close()

	logger := InitializeTelemetry(telemetryFile)
	slog.Info("setup", "status", "initialized telemetry successful")

	slog.Info("setup-telemetry", "location", telemetryFilePath)
	slog.Info("setup-environment", "config", configJson)

	// Database usage and connection
	db, err, usable := ConnectToDatabase(config.DatabaseConfig.Uri)
	if err != nil {
		slog.Warn("database-creation", "cause", "db file error", "reason", err)
	}
	if !usable {
		slog.Error("database-creation", "cause", "failure to use db")
		slog.Info("setup", "operation", "terminating")
		os.Exit(1)
	}

	// Database file backup worker handeling
	taskHandle := make(<-chan backup.TaskHandleSignal, 20)
	signalHandler, ticker := backup.CreateFileBackupTask(
		backup.BackupLocations{
			SourceLocation: config.DatabaseConfig.Uri,
			BackupLocation: config.DatabaseConfig.BackUpLocation,
		},
		taskHandle,
		config.DatabaseConfig.BackupInterval,
	)
	// TODO Handle task signaler
	_ = signalHandler
	defer ticker.Stop()

	// execute a query on the server
	err = RunMigrations(db, "./migrations/")
	if err != nil {
		slog.Error("setup-db-migrations", "cause", err.Error())
		slog.Error("setup-db-migrations", "cause", "Could not migrate db changes")
		os.Exit(1)
	}

	// Web App config and launch
	app := gin.Default()
	api := app.Group("/api")
	HttpApi(api)

	server := &http.Server{
		Handler:      app,
		Addr:         fmt.Sprintf(":%d", config.WebApiConfig.Port),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		ReadTimeout:  time.Duration(config.WebApiConfig.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WebApiConfig.WriteTimeout) * time.Second,
	}

	server.ListenAndServe()
}

func HttpApi(api *gin.RouterGroup) {
	v1 := api.Group("/v1")

	devices := v1.Group("/devices")

	// /v1/devices/pmd
	pmd := devices.Group("/pmd")

	// /v1/devices/pmd/data
	_ = pmd.Group("/data")

	// /v1/devices/pmd/status
	status := pmd.Group("/status")
	status.POST("/", func(c *gin.Context) {})
	status.GET("/", func(c *gin.Context) {})
}
