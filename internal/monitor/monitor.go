package monitor

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
)

type Controller interface {
	Status(ctx context.Context) (colima.Profile, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context, force bool) error
}

type EventSource interface {
	Watch(ctx context.Context, notify func()) error
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
	events     chan struct{}
	running    atomic.Bool
	latest     *colima.Profile

	eventDebounce time.Duration
	retryMinimum  time.Duration
	retryMaximum  time.Duration
}

func New(controller Controller, interval time.Duration, onState func(State)) *Monitor {
	return &Monitor{
		controller:    controller,
		interval:      interval,
		onState:       onState,
		actions:       make(chan Action, 1),
		events:        make(chan struct{}, 1),
		eventDebounce: 500 * time.Millisecond,
		retryMinimum:  time.Second,
		retryMaximum:  5 * time.Minute,
	}
}

func (monitor *Monitor) Run(ctx context.Context) {
	monitor.perform(ctx, ActionRefresh)
	ticker := time.NewTicker(monitor.interval)
	defer ticker.Stop()

	watchDone := make(chan struct{})
	if source, ok := monitor.controller.(EventSource); ok {
		go func() {
			defer close(watchDone)
			monitor.watch(ctx, source)
		}()
	} else {
		close(watchDone)
	}
	defer func() { <-watchDone }()

	var eventTimer *time.Timer
	var eventTimerChannel <-chan time.Time
	defer func() {
		if eventTimer != nil {
			eventTimer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			monitor.perform(ctx, ActionRefresh)
		case action := <-monitor.actions:
			monitor.perform(ctx, action)
		case <-monitor.events:
			if eventTimer == nil {
				eventTimer = time.NewTimer(monitor.eventDebounce)
			} else {
				if !eventTimer.Stop() {
					select {
					case <-eventTimer.C:
					default:
					}
				}
				eventTimer.Reset(monitor.eventDebounce)
			}
			eventTimerChannel = eventTimer.C
		case <-eventTimerChannel:
			eventTimerChannel = nil
			monitor.perform(ctx, ActionRefresh)
		}
	}
}

func (monitor *Monitor) watch(ctx context.Context, source EventSource) {
	retryDelay := monitor.retryMinimum
	for {
		startedAt := time.Now()
		err := source.Watch(ctx, monitor.notifyEvent)
		if ctx.Err() != nil || errors.Is(err, colima.ErrWatchUnsupported) {
			return
		}
		if time.Since(startedAt) >= monitor.retryMaximum {
			retryDelay = monitor.retryMinimum
		}

		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		retryDelay *= 2
		if retryDelay > monitor.retryMaximum {
			retryDelay = monitor.retryMaximum
		}
	}
}

func (monitor *Monitor) notifyEvent() {
	select {
	case monitor.events <- struct{}{}:
	default:
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
