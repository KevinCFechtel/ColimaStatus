// Package monitor serializes Colima status checks and actions. It combines
// a safety interval, debounced Lima lifecycle events, and bounded retry so
// that no two operations ever overlap.
package monitor

import (
	"context"
	"errors"
	"log"
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
	// Watching reports whether Lima lifecycle events are currently being
	// received. When it is false the safety interval is the only source of
	// updates, which the menu says out loud rather than leaving the user to
	// wonder why a change took minutes to appear.
	Watching bool
}

type Monitor struct {
	controller Controller
	interval   time.Duration
	onState    func(State)
	actions    chan Action
	events     chan struct{}
	wake       chan struct{}
	running    atomic.Bool
	watching   atomic.Bool
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
		wake:          make(chan struct{}, 1),
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
		monitor.watching.Store(true)
		err := source.Watch(ctx, monitor.notifyEvent)
		monitor.watching.Store(false)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, colima.ErrWatchUnsupported) {
			log.Printf("Lima event watching is unavailable, falling back to periodic checks: %v", err)
			return
		}
		log.Printf("Lima event stream ended, retrying: %v", err)
		if time.Since(startedAt) >= monitor.retryMaximum {
			retryDelay = monitor.retryMinimum
		}

		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-monitor.wake:
			// Waking is the most likely moment for the stream to become
			// available again, so the accumulated backoff is discarded.
			timer.Stop()
			retryDelay = monitor.retryMinimum
			continue
		case <-timer.C:
		}
		retryDelay *= 2
		if retryDelay > monitor.retryMaximum {
			retryDelay = monitor.retryMaximum
		}
	}
}

// NotifyWake reports that the machine came back from sleep. It refreshes the
// status, because timers did not advance while the machine was asleep and the
// displayed state is as old as the moment the lid was closed, and it releases
// the watcher from its retry delay: an event stream that died during sleep
// would otherwise stay down for up to the maximum backoff.
func (monitor *Monitor) NotifyWake() {
	select {
	case monitor.wake <- struct{}{}:
	default:
	}
	monitor.Trigger(ActionRefresh)
}

func (monitor *Monitor) notifyEvent() {
	select {
	case monitor.events <- struct{}{}:
	default:
	}
}

// Trigger requests an action. A request that arrives while another operation
// is running stays in the queue and is performed afterwards instead of being
// dropped, so that a click during a long start is not silently lost. The queue
// holds one action: a second request while one is already waiting is redundant.
func (monitor *Monitor) Trigger(action Action) {
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

	monitor.onState(State{Profile: monitor.latest, Busy: action, Watching: monitor.watching.Load()})
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
	monitor.onState(State{Profile: monitor.latest, Err: err, Watching: monitor.watching.Load()})
}
