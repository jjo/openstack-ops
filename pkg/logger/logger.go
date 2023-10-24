package logger

import (
	"os"

	logging "github.com/op/go-logging"
)

var (
	Log    *logging.Logger
	Module = "os-cleanup"
)

func setupLogging(module string, out *os.File, logLevel string) {
	Log = logging.MustGetLogger(module)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000}: %{level:.6s} %{id:03x}%{color:reset} %{message}`,
	)
	backend := logging.NewLogBackend(out, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)
}

func SetLevel(logLevel string) *logging.Logger {
	level, err := logging.LogLevel(logLevel)
	if err != nil {
		Log.Fatalf("Invalid log level: %s", logLevel)
	}
	logging.SetLevel(level, Module)
	return Log
}

func init() {
	setupLogging(Module, os.Stderr, "info")
}
