package monitor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
)

type fakeController struct {
	mu         sync.Mutex
	profile    colima.Profile
	statusCall int
	startCall  int
	stopForce  []bool
}

func (controller *fakeController) Status(context.Context) (colima.Profile, error) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.statusCall++
	return controller.profile, nil
}

type eventController struct {
	*fakeController
	events     chan struct{}
	watchCalls int
}

type retryEventController struct {
	*fakeController
	secondWatchStarted chan struct{}
	watchCalls         int
}

func (controller *retryEventController) Watch(ctx context.Context, _ func()) error {
	controller.mu.Lock()
	controller.watchCalls++
	call := controller.watchCalls
	controller.mu.Unlock()
	if call == 1 {
		return errors.New("temporary watch failure")
	}
	if call == 2 {
		close(controller.secondWatchStarted)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (controller *eventController) Watch(ctx context.Context, notify func()) error {
	controller.mu.Lock()
	controller.watchCalls++
	controller.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-controller.events:
			notify()
		}
	}
}

func (controller *fakeController) Start(context.Context) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.startCall++
	controller.profile.State = colima.StateRunning
	return nil
}

func (controller *fakeController) Stop(_ context.Context, force bool) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.stopForce = append(controller.stopForce, force)
	controller.profile.State = colima.StateStopped
	return nil
}

func TestMonitorRefreshesAndStarts(t *testing.T) {
	controller := &fakeController{profile: colima.Profile{Name: "default", State: colima.StateStopped}}
	states := make(chan State, 8)
	monitor := New(controller, time.Hour, func(state State) { states <- state })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		monitor.Run(ctx)
	}()

	waitForIdleState(t, states, colima.StateStopped)
	monitor.Trigger(ActionStart)
	waitForIdleState(t, states, colima.StateRunning)

	cancel()
	<-done
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.startCall != 1 {
		t.Fatalf("Start() calls = %d, want 1", controller.startCall)
	}
}

func TestMonitorForceStopsBrokenProfile(t *testing.T) {
	controller := &fakeController{profile: colima.Profile{Name: "default", State: colima.StateBroken}}
	states := make(chan State, 8)
	monitor := New(controller, time.Hour, func(state State) { states <- state })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		monitor.Run(ctx)
	}()

	waitForIdleState(t, states, colima.StateBroken)
	monitor.Trigger(ActionStop)
	waitForIdleState(t, states, colima.StateStopped)
	cancel()
	<-done

	controller.mu.Lock()
	defer controller.mu.Unlock()
	if len(controller.stopForce) != 1 || !controller.stopForce[0] {
		t.Fatalf("Stop() force values = %#v, want [true]", controller.stopForce)
	}
}

func TestMonitorDebouncesLifecycleEvents(t *testing.T) {
	base := &fakeController{profile: colima.Profile{Name: "default", State: colima.StateStopped}}
	controller := &eventController{fakeController: base, events: make(chan struct{}, 3)}
	states := make(chan State, 8)
	monitor := New(controller, time.Hour, func(state State) { states <- state })
	monitor.eventDebounce = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		monitor.Run(ctx)
	}()

	waitForIdleState(t, states, colima.StateStopped)
	controller.mu.Lock()
	controller.profile.State = colima.StateRunning
	controller.mu.Unlock()
	controller.events <- struct{}{}
	controller.events <- struct{}{}
	controller.events <- struct{}{}
	waitForIdleState(t, states, colima.StateRunning)

	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.statusCall != 2 {
		t.Fatalf("Status() calls = %d, want initial check plus one debounced event check", controller.statusCall)
	}
	if controller.watchCalls != 1 {
		t.Fatalf("Watch() calls = %d, want 1", controller.watchCalls)
	}
}

func TestMonitorRestartsFailedEventSource(t *testing.T) {
	base := &fakeController{profile: colima.Profile{Name: "default", State: colima.StateStopped}}
	controller := &retryEventController{
		fakeController:     base,
		secondWatchStarted: make(chan struct{}),
	}
	states := make(chan State, 4)
	monitor := New(controller, time.Hour, func(state State) { states <- state })
	monitor.retryMinimum = time.Millisecond
	monitor.retryMaximum = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		monitor.Run(ctx)
	}()

	waitForIdleState(t, states, colima.StateStopped)
	select {
	case <-controller.secondWatchStarted:
	case <-time.After(time.Second):
		t.Fatal("event source was not restarted")
	}
	cancel()
	<-done

	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.watchCalls != 2 {
		t.Fatalf("Watch() calls = %d, want 2", controller.watchCalls)
	}
}

func waitForIdleState(t *testing.T, states <-chan State, want colima.State) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case state := <-states:
			if state.Busy == "" && state.Profile != nil && state.Profile.State == want {
				return
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for idle state %q", want)
		}
	}
}

// A click that arrives while an operation is running must not be lost. The
// action waits in the queue and runs afterwards, which is what a user expects
// after pressing refresh during a long start.
func TestTriggerQueuesAnActionArrivingWhileBusy(t *testing.T) {
	t.Parallel()

	released := make(chan struct{})
	controller := &blockingController{released: released}
	states := make(chan State, 16)
	testMonitor := New(controller, time.Hour, func(state State) { states <- state })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		testMonitor.Run(ctx)
	}()

	// The initial refresh is in flight and blocked inside Status.
	waitFor(t, func() bool { return controller.statusCalls() == 1 })
	testMonitor.Trigger(ActionRefresh)
	close(released)

	waitFor(t, func() bool { return controller.statusCalls() >= 2 })
	cancel()
	<-done
}

type blockingController struct {
	released chan struct{}
	blocked  atomic.Bool
	calls    atomic.Int64
}

func (controller *blockingController) Status(ctx context.Context) (colima.Profile, error) {
	controller.calls.Add(1)
	// Only the first call blocks; afterwards the queued action must get through.
	if controller.blocked.CompareAndSwap(false, true) {
		select {
		case <-controller.released:
		case <-ctx.Done():
			return colima.Profile{}, ctx.Err()
		}
	}
	return colima.Profile{Name: "default", State: colima.StateStopped}, nil
}

func (controller *blockingController) Start(context.Context) error { return nil }

func (controller *blockingController) Stop(context.Context, bool) error { return nil }

func (controller *blockingController) statusCalls() int64 { return controller.calls.Load() }

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for the expected state")
}
