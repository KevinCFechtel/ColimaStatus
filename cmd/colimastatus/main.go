package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"fyne.io/systray"

	"github.com/KevinCFechtel/ColimaStatus/internal/autostart"
	"github.com/KevinCFechtel/ColimaStatus/internal/buildinfo"
	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
	"github.com/KevinCFechtel/ColimaStatus/internal/config"
	"github.com/KevinCFechtel/ColimaStatus/internal/localization"
	"github.com/KevinCFechtel/ColimaStatus/internal/logging"
	"github.com/KevinCFechtel/ColimaStatus/internal/monitor"
	trayui "github.com/KevinCFechtel/ColimaStatus/internal/tray"
)

func main() {
	// Only two flags exist, so this stays hand-rolled rather than pulling in
	// the flag package for an app whose real interface is the menu. An
	// unrecognized argument is reported instead of silently starting the app
	// with settings the user believes they changed.
	for _, argument := range os.Args[1:] {
		switch argument {
		case "--version", "-v":
			fmt.Printf("ColimaStatus %s\n", buildinfo.Summary())
			return
		case "--help", "-h":
			printUsage(os.Stdout)
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown argument: %s\n\n", argument)
			printUsage(os.Stderr)
			os.Exit(2)
		}
	}
	os.Exit(run())
}

func printUsage(target io.Writer) {
	// Help output going nowhere is not worth reacting to.
	_, _ = fmt.Fprintf(target, `ColimaStatus %s

A macOS menu bar app for a local Colima installation. It runs without a Dock
icon; all interaction happens through the menu bar item.

Usage:
  ColimaStatus [--version] [--help]

Options:
  -h, --help     Show this help and exit.
  -v, --version  Show the version and exit.

Settings (edit the file; changes take effect on the next start):
  %s

Log:
  %s

Environment variables override the settings file. They are meant for running
the binary from a terminal: an app launched from Finder or as a login item
inherits no shell environment, which is why the settings file exists.
  COLIMASTATUS_PROFILE       Colima profile to monitor
  COLIMASTATUS_COLIMA_PATH   Colima executable in a non-standard location
  COLIMASTATUS_LANGUAGE      Force "en" or "de" instead of following macOS
`, buildinfo.Summary(), config.DescribePath(), logging.DescribePath())
}

// run owns the exit code so that the log is closed on every path. Exiting from
// inside main would skip the deferred close and drop the very lines that
// explain why the start failed.
func run() int {
	closeLog := logging.Setup()
	defer closeLog()
	log.Printf("starting ColimaStatus %s", buildinfo.Summary())

	configuration, configurationPath := loadConfiguration()

	texts, err := localization.NewDetected(configuration.Language)
	if err != nil {
		log.Printf("localization could not be initialized: %v", err)
		return 1
	}

	app := trayui.New(trayui.Options{
		Controller: newController(configuration),
		Interval:   configuration.CheckInterval,
		Autostart:  autostart.NewNativeController(),
		Texts:      texts,
		LogPath:    logging.DescribePath(),
		ConfigPath: configurationPath,
	})
	systray.Run(app.OnReady, app.OnExit)
	return 0
}

// loadConfiguration reads the settings file and layers the documented
// environment variables on top. Both the effective settings and the path are
// returned, because the menu offers to reveal the file.
func loadConfiguration() (config.Config, string) {
	path, err := config.DefaultPath()
	if err != nil {
		log.Printf("configuration location could not be determined, using defaults: %v", err)
		return config.ApplyEnvironment(config.Default(), os.Getenv), config.DescribePath()
	}

	configuration, notes := config.Load(path)
	for _, note := range notes {
		log.Printf("configuration: %s", note)
	}
	configuration = config.ApplyEnvironment(configuration, os.Getenv)
	log.Printf("monitoring profile %q, safety interval %s, settings at %s",
		configuration.Profile, configuration.CheckInterval, path)
	return configuration, path
}

func newController(configuration config.Config) monitor.Controller {
	colimaPath, err := colima.Locate(configuration.ColimaPath)
	if err != nil {
		log.Printf("Colima could not be located: %v", err)
		return unavailableController{err: colima.Unavailable(err)}
	}
	log.Printf("using Colima at %s", colimaPath)
	client := colima.NewClient(colimaPath, configuration.Profile)
	versions := client.Versions(context.Background())
	if versions.Colima != "" {
		log.Printf("detected %s", versions.Colima)
	}
	if versions.Lima != "" {
		log.Printf("detected %s", versions.Lima)
	}
	return client
}

type unavailableController struct {
	err error
}

func (controller unavailableController) Status(context.Context) (colima.Profile, error) {
	return colima.Profile{}, controller.err
}

func (controller unavailableController) Start(context.Context) error {
	return controller.err
}

func (controller unavailableController) Stop(context.Context, bool) error {
	return controller.err
}
