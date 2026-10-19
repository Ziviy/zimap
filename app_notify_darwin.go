//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework AppKit -framework UserNotifications
#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <UserNotifications/UserNotifications.h>

// Delegate so banners are shown even when the app is frontmost (macOS suppresses
// foreground notifications when no UNUserNotificationCenterDelegate is set).
API_AVAILABLE(macos(10.14))
@interface ZimapNotifyDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

API_AVAILABLE(macos(10.14))
@implementation ZimapNotifyDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completionHandler {
	if (@available(macOS 11.0, *)) {
		completionHandler(UNNotificationPresentationOptionList | UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionSound);
	} else {
		completionHandler((1 << 2) | UNNotificationPresentationOptionSound);
	}
}
@end

// zimapPostNotification delivers a UNUserNotification attributed to the running
// app bundle, so macOS banners show the ZiMap icon (UNUserNotificationCenter
// always renders the posting bundle's icon; the deprecated NSUserNotification
// API did not reliably do so). Fire-and-forget: authorization is requested on
// first use and the request is enqueued from the completion handler, so the
// first-run permission prompt never blocks the caller. Returns NO when not
// running from a .app bundle (dev runs, tests), letting the caller fall back
// to osascript.
int zimapPostNotification(const char *title, const char *message) {
	@autoreleasepool {
		NSBundle *bundle = [NSBundle mainBundle];
		if (bundle == nil || bundle.bundleIdentifier == nil) {
			return NO;
		}

		if (@available(macOS 10.14, *)) {
			UNUserNotificationCenter *center = nil;
			@try {
				center = [UNUserNotificationCenter currentNotificationCenter];
			} @catch (NSException *exception) {
				center = nil;
			}
			if (center == nil) {
				return NO;
			}

			static ZimapNotifyDelegate *delegate = nil;
			static dispatch_once_t onceToken;
			dispatch_once(&onceToken, ^{
				delegate = [[ZimapNotifyDelegate alloc] init];
			});
			center.delegate = delegate;

			UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
			content.title = [NSString stringWithUTF8String:title];
			content.body = [NSString stringWithUTF8String:message];
			content.sound = [UNNotificationSound defaultSound];
			UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:[[NSUUID UUID] UUIDString] content:content trigger:nil];

			[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
			                      completionHandler:^(BOOL granted, NSError *error) {
				if (granted) {
					[center addNotificationRequest:request withCompletionHandler:nil];
				}
			}];
			return 1;
		}

		return NO;
	}
}
*/
import "C"
import "unsafe"

// notifyDarwin posts a native macOS notification via UNUserNotificationCenter,
// attributed to the app bundle (com.ziviy.zimap), so the banner shows the ZiMap
// icon. It returns false when the process is not running from a .app bundle and
// osascript should be used.
func notifyDarwin(title, message string) bool {
	titleC := C.CString(title)
	defer C.free(unsafe.Pointer(titleC))
	messageC := C.CString(message)
	defer C.free(unsafe.Pointer(messageC))
	return C.zimapPostNotification(titleC, messageC) == 1
}
