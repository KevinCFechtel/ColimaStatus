package colima

import "time"

type State string

const (
	StateMissing State = "missing"
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateBroken  State = "broken"
	StateUnknown State = "unknown"
)

type Profile struct {
	Name      string
	State     State
	RawStatus string
	Arch      string
	CPUs      int
	Memory    int64
	Disk      int64
	Runtime   string
	Address   string
	CheckedAt time.Time
}
