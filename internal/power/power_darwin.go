//go:build darwin

package power

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation

void ColimaStatusStartWakeObserver(void);
void ColimaStatusStopWakeObserver(void);
*/
import "C"

import "sync"

// The observer is process-wide because NSWorkspace's notification centre is.
// A second Watch replaces the callback rather than adding another observer.
var (
	mutex    sync.Mutex
	onWake   func()
	observed bool
)

func watch(notify func()) (stop func()) {
	mutex.Lock()
	onWake = notify
	alreadyObserved := observed
	observed = true
	mutex.Unlock()

	if !alreadyObserved {
		C.ColimaStatusStartWakeObserver()
	}
	return func() {
		C.ColimaStatusStopWakeObserver()
		mutex.Lock()
		onWake = nil
		observed = false
		mutex.Unlock()
	}
}

// ColimaStatusNotifyWake is called from the Objective-C observer block. It runs
// on the main thread, so it only forwards the event and returns.
//
//export ColimaStatusNotifyWake
func ColimaStatusNotifyWake() {
	mutex.Lock()
	notify := onWake
	mutex.Unlock()
	if notify != nil {
		notify()
	}
}
