package buildinfo

import "testing"

func TestSummary(t *testing.T) {
	t.Parallel()

	if got := Summary(); got != "dev (build 0, commit unknown)" {
		t.Fatalf("Summary() = %q", got)
	}
}
