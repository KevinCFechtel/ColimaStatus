//go:build windows

package localization

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const localeNameMaximumLength = 85

var getUserDefaultLocaleName = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")

func platformPreferredLanguages() []string {
	buffer := make([]uint16, localeNameMaximumLength)
	written, _, _ := getUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if written == 0 {
		return environmentPreferredLanguages(os.Getenv)
	}
	return []string{windows.UTF16ToString(buffer)}
}
