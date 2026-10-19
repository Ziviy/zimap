//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit
#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>

// zimapPostNotification delivers a notification attributed to the running app
// bundle, so macOS banners show the ZiMap icon instead of the Script Editor
// paper icon. Returns NO when not running from a .app bundle (dev runs, tests),
// letting the caller fall back to osascript.
int zimapPostNotification(const char *title, const char *message) {
	NSBundle *bundle = [NSBundle mainBundle];
	if (bundle == nil || bundle.bundleIdentifier == nil) {
		return NO;
	}

	NSUserNotification *notification = [[NSUserNotification alloc] init];
	notification.title = [NSString stringWithUTF8String:title];
	notification.informativeText = [NSString stringWithUTF8String:message];
	notification.soundName = NSUserNotificationDefaultSoundName;
	[[NSUserNotificationCenter defaultUserNotificationCenter] deliverNotification:notification];
	return 1;
}
*/
import "C"
import "unsafe"

// notifyDarwin posts a native macOS notification attributed to the app bundle
// (com.ziviy.zimap), so the banner shows the ZiMap icon. It returns false when
// the process is not running from a .app bundle and osascript should be used.
func notifyDarwin(title, message string) bool {
	titleC := C.CString(title)
	defer C.free(unsafe.Pointer(titleC))
	messageC := C.CString(message)
	defer C.free(unsafe.Pointer(messageC))
	return C.zimapPostNotification(titleC, messageC) == 1
}
