#import <AppKit/AppKit.h>

// Implemented in Go via //export.
extern void ColimaStatusNotifyWake(void);

static id wakeObserver = nil;

void ColimaStatusStartWakeObserver(void)
{
    if (wakeObserver != nil) {
        return;
    }
    // NSWorkspace has its own notification centre; the default NSNotificationCenter
    // never receives NSWorkspaceDidWakeNotification.
    wakeObserver = [[[NSWorkspace sharedWorkspace] notificationCenter]
        addObserverForName:NSWorkspaceDidWakeNotification
                    object:nil
                     queue:nil
                usingBlock:^(NSNotification *notification) {
                    (void)notification;
                    ColimaStatusNotifyWake();
                }];
}

void ColimaStatusStopWakeObserver(void)
{
    if (wakeObserver == nil) {
        return;
    }
    [[[NSWorkspace sharedWorkspace] notificationCenter] removeObserver:wakeObserver];
    wakeObserver = nil;
}
