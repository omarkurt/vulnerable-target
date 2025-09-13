// Package logger provides logging functionality for the vulnerable target application.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitWithLevel initializes the logger with the specified verbosity level.
func InitWithLevel(verbosityLevel string) {
	level, err := zerolog.ParseLevel(verbosityLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.TimeOnly,
		NoColor:    false,
	}

	log.Logger = zerolog.
		New(output).
		Level(level).
		With().
		Timestamp().
		Logger()
}

// Init initializes the logger with the default info level.
func Init() {
	InitWithLevel("info")
}
