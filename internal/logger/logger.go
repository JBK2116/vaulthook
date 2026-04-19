// Package logger configures and exposes the application-wide structured logger.
//
// The logger is initialized once and passed throughout the application.
// It is backed by zerolog with console output, structured timestamps,
// and stack trace support for error reporting.
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

// once guards logger initialization so it occurs exactly once per process.
var once sync.Once

// logger is the package-level singleton instance.
var logger zerolog.Logger

// NewLogger constructs and returns a singleton zerolog.Logger configured with
// console output, RFC3339Nano timestamps, and stack trace marshaling.
//
// The logger is initialized exactly once via sync.Once. Subsequent calls
// return the same instance. The log level is sourced from config.Envs.LOG_LEVEL.
//
// Each log entry includes the git revision and Go version extracted from
// the binary's embedded build info. If build info is unavailable, an error
// is returned and no logger is produced.
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
		logger = zerolog.New(output).Level(zerolog.Level(config.Envs.LogLevel)).With().Timestamp().Str("git_revision", gitRevision).Str("go_version", goVersion).Logger()
	})
	return &logger, err
}
