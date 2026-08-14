package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
)

type fakeController struct {
	mu        sync.Mutex
	profile   colima.Profile
	startCall int
	stopForce []bool
}

func (controller *fakeController) Status(context.Context) (colima.Profile, error) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return controller.profile, nil
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
