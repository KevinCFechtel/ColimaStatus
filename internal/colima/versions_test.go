package colima

import (
	"context"
	"testing"
)

func TestVersionsAreDiagnosticOnly(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{
		outputs: []CommandOutput{
			{Stdout: "colima version 0.10.3\ngit commit: abc\n"},
			{Stdout: "limactl version 2.2.0\n"},
		},
	}
	client := NewClient("/opt/colima", "default")
	client.limaPath = "/opt/limactl"
	client.runner = runner

	got := client.Versions(context.Background())
	if got.Colima != "colima version 0.10.3" || got.Lima != "limactl version 2.2.0" {
		t.Fatalf("Versions() = %#v", got)
	}
}

func TestVersionsIgnoreMissingLimaAndCommandFailure(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{errors: []error{context.DeadlineExceeded}}
	client := NewClient("/opt/colima", "default")
	client.limaPath = ""
	client.runner = runner

	got := client.Versions(context.Background())
	if got.Colima != "" || got.Lima != "" {
		t.Fatalf("Versions() = %#v, want empty diagnostic values", got)
	}
}
