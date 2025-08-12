package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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

func Init() {
	InitWithLevel("info")
}
