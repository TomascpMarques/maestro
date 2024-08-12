package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	gin "github.com/gin-gonic/gin"
	validator "github.com/go-playground/validator/v10"
	_ "github.com/mattn/go-sqlite3"
)

// Struct validation across the entire app
var VALIDATE *validator.Validate = validator.New(
	validator.WithRequiredStructEnabled(),
	validator.WithPrivateFieldValidation(),
)

/*
	// TODO: Create Database according to env
	TODO: Backup Database and zip it according to env
*/

func main() {
	configPath, defined := os.LookupEnv("ENV_PATH")
	if !defined {
		log.Fatalf("No Environment file specified")
	}
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Config Error:\n%s\n", err.Error())
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		panic("Should not fail to parse config to json!")
	}

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

	db, err, usable := ConnectToDatabase(config.DatabaseConfig.Uri)
	if err != nil {
		slog.Warn("database-creation", "cause", "db file error", "reason", err)
	}
	if !usable {
		slog.Error("database-creation", "cause", "failure to use db")
		slog.Info("setup", "operation", "terminating")
		os.Exit(1)
	}

	schema := `CREATE TABLE spo (
	    value integer,
	    time text
	    );`

	// execute a query on the server
	_, err = db.Exec(schema)
	if err != nil {
		slog.Error("setup-db", "cause", "Could not build")
	}

	app := gin.Default()

	api := app.Group("/api")
	HttpApi(api)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      app,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	server.ListenAndServe()
}

func HttpApi(api *gin.RouterGroup) {
	v1 := api.Group("/v1")

	dataSources := v1.Group("/source")

	pmds := dataSources.Group("/pmd")

	pmds.POST("/spo2", func(ctx *gin.Context) {
		type Spo2Measure struct {
			Value      int    `form:"value"  binding:"required"`
			SourceID   string `form:"source" binding:"required"`
			DeviceType string `form:"device" binding:"required"`
		}

		var spo2Measure Spo2Measure
		if err := ctx.ShouldBind(&spo2Measure); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid data",
			})
			log.Printf("Erro: %s", err.Error())
			return
		}

		ctx.Status(http.StatusAccepted)
	})
}
