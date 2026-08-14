package main

import (
	"context"
	"errors"
	"os"
	"time"

	"fyne.io/systray"

	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
	"github.com/KevinCFechtel/ColimaStatus/internal/monitor"
	trayui "github.com/KevinCFechtel/ColimaStatus/internal/tray"
)

const defaultCheckInterval = time.Minute

func main() {
	app := trayui.New(newController(), defaultCheckInterval)
	systray.Run(app.OnReady, app.OnExit)
}

func newController() monitor.Controller {
	colimaPath, err := colima.Locate(os.Getenv("COLIMASTATUS_COLIMA_PATH"))
	if err != nil {
		return unavailableController{err: errors.New("Colima wurde nicht gefunden")}
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
