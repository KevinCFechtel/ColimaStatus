package monitor

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
)

type Controller interface {
	Status(ctx context.Context) (colima.Profile, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context, force bool) error
}

type Action string

const (
	ActionRefresh Action = "refresh"
	ActionStart   Action = "start"
	ActionStop    Action = "stop"
)

type State struct {
	Profile *colima.Profile
	Busy    Action
	Err     error
}

type Monitor struct {
	controller Controller
	interval   time.Duration
	onState    func(State)
	actions    chan Action
	running    atomic.Bool
	latest     *colima.Profile
}

func New(controller Controller, interval time.Duration, onState func(State)) *Monitor {
	return &Monitor{
		controller: controller,
		interval:   interval,
		onState:    onState,
		actions:    make(chan Action, 1),
	}
}

func (monitor *Monitor) Run(ctx context.Context) {
	monitor.perform(ctx, ActionRefresh)
	ticker := time.NewTicker(monitor.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			monitor.perform(ctx, ActionRefresh)
		case action := <-monitor.actions:
			monitor.perform(ctx, action)
		}
	}
}

func (monitor *Monitor) Trigger(action Action) {
	if monitor.running.Load() {
		return
	}
	select {
	case monitor.actions <- action:
	default:
	}
}

func (monitor *Monitor) perform(ctx context.Context, action Action) {
	if !monitor.running.CompareAndSwap(false, true) {
		return
	}
	defer monitor.running.Store(false)

	monitor.onState(State{Profile: monitor.latest, Busy: action})
	var actionErr error
	switch action {
	case ActionStart:
		actionErr = monitor.controller.Start(ctx)
	case ActionStop:
		force := monitor.latest != nil && monitor.latest.State == colima.StateBroken
		actionErr = monitor.controller.Stop(ctx, force)
	}

	profile, statusErr := monitor.controller.Status(ctx)
	if statusErr == nil {
		monitor.latest = &profile
	}
	err := actionErr
	if err == nil {
		err = statusErr
	}
	monitor.onState(State{Profile: monitor.latest, Err: err})
}
