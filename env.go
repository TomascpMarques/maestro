package main

import (
	"errors"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	// Got to use V1, V2 will break trying to read time.Duration values
	toml "github.com/pelletier/go-toml"
)

// Wraps all the wanted configs in on place
type ConfigWrapper struct {
	DatabaseConfig  Database  `toml:"database" validate:"required"`
	WebApiConfig    WebApi    `toml:"web_api" validate:"required"`
	TelemetryConfig Telemetry `toml:"telemetry" validate:"required"`
}

type Database struct {
	Uri            string        `toml:"uri" validate:"required"`
	Backup         bool          `toml:"backup"`
	BackupInterval time.Duration `toml:"backup_interval" validate:"required"`
	BackUpLocation string        `toml:"location" validate:"required"`
}

type WebApi struct {
	Port         uint16 `toml:"port" validate:"required,gte=2000,lte=65535"`
	ReadTimeout  uint8  `toml:"read_timeout" validate:"required,gte=2,lte=1000"`
	WriteTimeout uint8  `toml:"write_timeout" validate:"required,gte=2,lte=1000"`
}

type Telemetry struct {
	Destination string `toml:"destination" validate:"required"`
}

/*
LoadConfig loads the config values from an env file, with the files
written using the TOML format.
*/
func LoadConfig(absOrigin string) (ConfigWrapper, error) {
	var config ConfigWrapper

	configFile, err := os.Open(absOrigin)
	if err != nil {
		return ConfigWrapper{},
			errors.New("that file either does not exist, or the path is wrong")
	}

	err = toml.NewDecoder(configFile).Decode(&config)
	if err != nil {
		return ConfigWrapper{},
			errors.New("failed to read config toml from: " + absOrigin)
	}

	err = VALIDATE.Struct(&config)
	if err != nil {
		validationErrors := []error{}
		for _, err := range err.(validator.ValidationErrors) {
			/*
				Here the erro var can be nil, because each ErrorMapper,
				and the subsequent append call, will always ensure that "e" has a value
				or is given one.
			*/
			var e error = nil
			WebApiEnvErrorMapper(err, &e)
			DatabaseEnvErrorMapper(err, &e)
			validationErrors = append(validationErrors, e)
		}
		return config, errors.Join(validationErrors...)
	}

	return config, nil
}

/*
WebApiEnvErrorMapper Maps a validator error field into the expected
error value, is used to customize the error messages given an invalid
config field value.
*/
func WebApiEnvErrorMapper(err validator.FieldError, e *error) {
	switch err.Field() {
	case "Port":
		*e = errors.New("PORT number should be between 1000 and 65535")
	case "ReadTimeout":
		*e = errors.New("READ-TIMEOUT should be between 2 and 1000")
	case "WriteTimeout":
		*e = errors.New("READ-TIMEOUT should be between 2 and 1000")
	default:
		return
	}
}

func DatabaseEnvErrorMapper(err validator.FieldError, e *error) {
	switch err.Field() {
	case "BackUpLocation":
		*e = errors.New("BACKUP-LOCATION should be a file path to store the DB backup")
	case "Uri":
		*e = errors.New("URI should be a file path to store the DB")
	default:
		return
	}
}
