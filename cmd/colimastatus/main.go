package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"fyne.io/systray"

	"github.com/KevinCFechtel/ColimaStatus/internal/autostart"
	"github.com/KevinCFechtel/ColimaStatus/internal/buildinfo"
	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
	"github.com/KevinCFechtel/ColimaStatus/internal/localization"
	"github.com/KevinCFechtel/ColimaStatus/internal/monitor"
	trayui "github.com/KevinCFechtel/ColimaStatus/internal/tray"
)

const fallbackCheckInterval = 15 * time.Minute

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("ColimaStatus %s\n", buildinfo.Summary())
		return
	}
	log.Printf("starting ColimaStatus %s", buildinfo.Summary())

	texts, err := localization.NewDetected()
	if err != nil {
		log.Fatalf("localization could not be initialized: %v", err)
	}

	autostartController := autostart.NewNativeController()
	app := trayui.New(newController(texts), fallbackCheckInterval, autostartController, texts)
	systray.Run(app.OnReady, app.OnExit)
}

func newController(texts *localization.Strings) monitor.Controller {
	colimaPath, err := colima.Locate(os.Getenv("COLIMASTATUS_COLIMA_PATH"))
	if err != nil {
		return unavailableController{err: errors.New(texts.ColimaNotFound())}
	}

	profile := os.Getenv("COLIMASTATUS_PROFILE")
	return colima.NewClient(colimaPath, profile)
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
