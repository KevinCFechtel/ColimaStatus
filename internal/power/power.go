// Package power reports when macOS returns from sleep.
//
// It exists because Go's timers do not advance while the machine is asleep:
// the runtime reads time from mach_absolute_time, which is suspended along
// with the system. A closed laptop therefore does not run the safety check it
// was scheduled for, and the menu shows whatever was true before the lid was
// closed. Waking is exactly the moment the user looks at the menu bar, so it
// is the moment the status has to be re-read.
package power

// Watch installs a wake observer and returns a function that removes it again.
// The notification arrives on the main thread, so notify must not block; it is
// expected to hand the event to a channel.
//
// On platforms without wake notifications Watch is a no-op, and the periodic
// check remains the only trigger.
func Watch(notify func()) (stop func()) {
	return watch(notify)
}
