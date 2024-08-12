package telemetry

import (
	"log/slog"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Telemetry error types

type TelemetryError struct {
	Cause string
}

func (te TelemetryError) Error() string {
	return te.Cause
}

var (
	ErrFailedToCreateLocalStore = TelemetryError{"could not create local sqlite instance"}
	ErrFailedToRegister         = TelemetryError{"could not register telemetry data"}
)

func InitTelemetry() (*slog.Logger, *TelemetryError) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
	)

	return logger, nil
}
