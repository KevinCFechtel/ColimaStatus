//go:build darwin && cgo

package localization

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

static char *ColimaStatusPreferredLanguages(void) {
	@autoreleasepool {
		NSArray<NSString *> *languages = [NSLocale preferredLanguages];
		NSString *joined = [languages componentsJoinedByString:@"\n"];
		return strdup([joined UTF8String]);
	}
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

func platformPreferredLanguages() []string {
	value := C.ColimaStatusPreferredLanguages()
	if value == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(value))
	return strings.Fields(C.GoString(value))
}
