// Package logger is responsible for logging related logic
//
// This package handles configuring the logger, format and destination.
// It exposes a logger object that is passed throughout the application.
package logger

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var once sync.Once
var logger zerolog.Logger

// NewLogger
func NewLogger() (*zerolog.Logger, error) {
	var err error
	once.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = time.RFC3339Nano
		var output io.Writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
		var gitRevision string
		var goVersion string
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			err = fmt.Errorf("unable to read buildInfo")
			return
		}
		goVersion = buildInfo.GoVersion
		for _, v := range buildInfo.Settings {
			if v.Key == "vcs.revision" {
				gitRevision = v.Value
				break
			}
		}
		logger = zerolog.New(output).Level(zerolog.Level(config.Envs.LOG_LEVEL)).With().Timestamp().Str("git_revision", gitRevision).Str("go_version", goVersion).Logger()
	})
	return &logger, err
}
