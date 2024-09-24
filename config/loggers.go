package config

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is aliased for ease of use
type Logger = zerolog.Logger

// GetLogger returns the main logger used throughout the application
func GetLogger(ctx string) zerolog.Logger {
	return log.With().Str("ctx", ctx).Logger()
}

func initMainLogger() (err error) {
	// create a log writer which rotates logs upon exceeding 25 MB
	writer := lumberjack.Logger{
		Filename:   path.Join(*Config.LogDir, "controller.log"),
		MaxSize:    *Config.MaxLogSizeMB,
		MaxBackups: *Config.MaxLogBackups,
	}
	// configure logger
	zerolog.TimestampFieldName = "ts"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = time.RFC3339Nano

	RefreshLogLevel()

	// TODO: can we reorder timestamp (might have to check zap)
	log = zerolog.New(&writer).With().Timestamp().Logger()
	return err
}

func InitStdoutLoggers() (err error) {
	if !*Config.DisableStdOutLogging {
		if err := stdoutLogger(); err != nil {
			return fmt.Errorf("error starting stdout logger routine: %s", err.Error())
		}
	}
	return nil
}

// Redirecting log to stdout, it can be visible from docker logs in prod container's stdout
// but not in dev container since dev container's entry point is /bin/sh
func stdoutLogger() (err error) {
	log.Info().Msg("stdout logger routine initiated")
	go func() {
		cmd := "kill $(ps aux | grep 'tail')" //Kill tail process from previous execution
		out := exec.Command("sh", "-c", cmd)
		_ = out.Run()
		fileName := path.Join(*Config.LogDir, "controller.log")
		cmd = "tail -f -n+1" + " " + fileName
		out = exec.Command("sh", "-c", cmd)
		out.Stdout = os.Stdout
		out.Stderr = os.Stderr
		if err = out.Run(); err != nil {
			log.Error().Msg(fmt.Sprintf("failed to execute command: %s", cmd))
			return
		}
	}()
	return nil
}

// RefreshLogLevel updates log level based on log-level input
func RefreshLogLevel() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	switch strings.ToLower(*Config.LogLevel) {
	case string(LogDebug):
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case string(LogTrace):
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
}

func initLoggers() (err error) {
	// hack to detect --cleanup flag even before it's parsed
	for _, s := range os.Args {
		if s == "--cleanup" {
			os.RemoveAll(*Config.LogDir)
			break
		}
		if s == "--no-stdout" || s == "-no-stdout" {
			*Config.DisableStdOutLogging = true
		}

	}

	if err = os.MkdirAll(*Config.LogDir, os.ModePerm); err != nil {
		return err
	}
	// this is needed because mkdirall permissions are not guaranteed
	if err := os.Chmod(*Config.LogDir, 0771); err != nil {
		return err
	}
	return initMainLogger()
}
