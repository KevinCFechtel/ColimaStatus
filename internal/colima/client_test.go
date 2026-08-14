package colima

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type runnerCall struct {
	executable string
	args       []string
}

type fakeRunner struct {
	outputs []CommandOutput
	errors  []error
	calls   []runnerCall
}

func (runner *fakeRunner) Run(_ context.Context, executable string, args ...string) (CommandOutput, error) {
	runner.calls = append(runner.calls, runnerCall{executable: executable, args: append([]string(nil), args...)})
	index := len(runner.calls) - 1
	var output CommandOutput
	if index < len(runner.outputs) {
		output = runner.outputs[index]
	}
	if index < len(runner.errors) {
		return output, runner.errors[index]
	}
	return output, nil
}

func TestClientStatusSelectsConfiguredProfile(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{outputs: []CommandOutput{{Stdout: `{"name":"default","status":"Stopped"}
{"name":"work","status":"Running","runtime":"docker"}`}}}
	checkedAt := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	client := NewClient("/custom/colima", "work")
	client.runner = runner
	client.now = func() time.Time { return checkedAt }

	profile, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if profile.Name != "work" || profile.State != StateRunning || !profile.CheckedAt.Equal(checkedAt) {
		t.Fatalf("Status() = %#v", profile)
	}
	wantCall := runnerCall{executable: "/custom/colima", args: []string{"list", "--json"}}
	if !reflect.DeepEqual(runner.calls[0], wantCall) {
		t.Fatalf("runner call = %#v, want %#v", runner.calls[0], wantCall)
	}
}

func TestClientStatusReturnsMissingProfile(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{outputs: []CommandOutput{{Stdout: `{"name":"other","status":"Running"}`}}}
	client := NewClient("colima", "default")
	client.runner = runner

	profile, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if profile.Name != "default" || profile.State != StateMissing {
		t.Fatalf("Status() = %#v", profile)
	}
}

func TestClientActionsUseProfileAndForce(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	client := NewClient("colima", "work")
	client.runner = runner

	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := client.Stop(context.Background(), true); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	want := []runnerCall{
		{executable: "colima", args: []string{"start", "work"}},
		{executable: "colima", args: []string{"stop", "work", "--force"}},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("runner calls = %#v, want %#v", runner.calls, want)
	}
}

func TestClientDefaultProfileOmitsProfileArgument(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	client := NewClient("colima", "")
	client.runner = runner

	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := client.Stop(context.Background(), false); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	want := []runnerCall{
		{executable: "colima", args: []string{"start"}},
		{executable: "colima", args: []string{"stop"}},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("runner calls = %#v, want %#v", runner.calls, want)
	}
}

func TestClientWrapsCommandError(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{errors: []error{errors.New("exit status 1")}}
	client := NewClient("colima", "")
	client.runner = runner

	if err := client.Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil, want command error")
	}
}
