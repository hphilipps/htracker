package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/exp/slog"
)

const envVarPrefix = "HTRACKER"

var (
	rootfs       = flag.NewFlagSet("root", flag.ExitOnError)
	logLevelFlag = rootfs.String("loglevel", "INFO", "log level (DEBUG|INFO|WARN|ERROR|OFF)")
)

func main() {
	ctx := context.Background()

	servecmd := &ffcli.Command{
		Name:       "serve",
		ShortUsage: "htracker <flags> serve <serve flags>",
		ShortHelp:  "start tracking sites and serving requests",
		LongHelp:   `The serve subcommand is starting up all components for tracking websites, notifying subscribers and listening to requests.`,
		FlagSet:    servefs,
		Exec:       newServeFunc(),
		Options:    []ff.Option{ff.WithEnvVarPrefix(envVarPrefix)},
	}

	rootcmd := ffcli.Command{
		Name:        "htracker",
		ShortUsage:  "htracker <flags> cmd <cmd_flags>",
		ShortHelp:   "htracker is a tool for tracking changes on websites",
		FlagSet:     rootfs,
		Subcommands: []*ffcli.Command{servecmd},
		Options:     []ff.Option{ff.WithEnvVarPrefix(envVarPrefix)},
	}

	if err := rootcmd.ParseAndRun(ctx, os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}

func createLogger(levelStr string) (*slog.Logger, error) {
	var lvl slog.Level

	switch levelStr {
	case slog.LevelDebug.String():
		lvl = slog.LevelDebug
	case slog.LevelInfo.String():
		lvl = slog.LevelInfo
	case slog.LevelWarn.String():
		lvl = slog.LevelWarn
	case slog.LevelError.String():
		lvl = slog.LevelError
	case "OFF":
		lvl = slog.Level(99)
	default:
		return nil, fmt.Errorf("log level %s not supported", *logLevelFlag)
	}
	return slog.New(slog.HandlerOptions{Level: lvl}.NewTextHandler(os.Stdout)), nil
}
