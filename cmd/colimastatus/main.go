package main

import (
	"context"
	"errors"
	"fmt"
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
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("ColimaStatus %s\n", buildinfo.Summary())
		return
	}
	os.Exit(run())
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
		Controller: newController(configuration, texts),
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

func newController(configuration config.Config, texts *localization.Strings) monitor.Controller {
	colimaPath, err := colima.Locate(configuration.ColimaPath)
	if err != nil {
		log.Printf("Colima could not be located: %v", err)
		return unavailableController{err: errors.New(texts.ColimaNotFound())}
	}
	log.Printf("using Colima at %s", colimaPath)
	return colima.NewClient(colimaPath, configuration.Profile)
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
