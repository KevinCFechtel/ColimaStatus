//go:build darwin

package autostart

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework ServiceManagement
#include <stdlib.h>

int ColimaStatusAutostartStatus(void);
int ColimaStatusSetAutostartEnabled(int enabled, char **errorMessage);
int ColimaStatusOpenAutostartSettings(void);
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

const nativeError = -1

type NativeController struct{}

func NewNativeController() Controller {
	return NativeController{}
}

func (NativeController) Status() (Status, error) {
	return statusFromNative(C.ColimaStatusAutostartStatus())
}

func (NativeController) SetEnabled(enabled bool) (Status, error) {
	nativeEnabled := C.int(0)
	if enabled {
		nativeEnabled = 1
	}

	var errorMessage *C.char
	nativeStatus := C.ColimaStatusSetAutostartEnabled(nativeEnabled, &errorMessage)
	if errorMessage != nil {
		defer C.free(unsafe.Pointer(errorMessage))
	}
	if nativeStatus == nativeError {
		if errorMessage == nil {
			return NotFound, errors.New("launch at login could not be changed")
		}
		return NotFound, errors.New(C.GoString(errorMessage))
	}
	return statusFromNative(nativeStatus)
}

func (NativeController) OpenSettings() error {
	if C.ColimaStatusOpenAutostartSettings() == 0 {
		return errors.New("Login Items settings require macOS 13 or later")
	}
	return nil
}

func statusFromNative(nativeStatus C.int) (Status, error) {
	switch nativeStatus {
	case 0:
		return Unsupported, nil
	case 1:
		return Disabled, nil
	case 2:
		return Enabled, nil
	case 3:
		return RequiresApproval, nil
	case 4:
		return NotFound, nil
	default:
		return NotFound, fmt.Errorf("unknown native launch at login status: %d", int(nativeStatus))
	}
}
