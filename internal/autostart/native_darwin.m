#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#include <string.h>

enum ColimaStatusAutostartStatus {
    ColimaStatusAutostartStatusError = -1,
    ColimaStatusAutostartStatusUnsupported = 0,
    ColimaStatusAutostartStatusDisabled = 1,
    ColimaStatusAutostartStatusEnabled = 2,
    ColimaStatusAutostartStatusRequiresApproval = 3,
    ColimaStatusAutostartStatusNotFound = 4,
};

static int ColimaStatusMapAutostartStatus(SMAppServiceStatus status)
API_AVAILABLE(macos(13.0))
{
    switch (status) {
        case SMAppServiceStatusNotRegistered:
            return ColimaStatusAutostartStatusDisabled;
        case SMAppServiceStatusEnabled:
            return ColimaStatusAutostartStatusEnabled;
        case SMAppServiceStatusRequiresApproval:
            return ColimaStatusAutostartStatusRequiresApproval;
        case SMAppServiceStatusNotFound:
            return ColimaStatusAutostartStatusNotFound;
    }
    return ColimaStatusAutostartStatusNotFound;
}

static void ColimaStatusSetErrorMessage(char **errorMessage, NSError *error)
{
    if (errorMessage == NULL) {
        return;
    }
    NSString *message = error.localizedDescription;
    if (message == nil || message.length == 0) {
        message = @"Autostart konnte nicht geändert werden";
    }
    *errorMessage = strdup(message.UTF8String);
}

int ColimaStatusAutostartStatus(void)
{
    if (@available(macOS 13.0, *)) {
        return ColimaStatusMapAutostartStatus(SMAppService.mainAppService.status);
    }
    return ColimaStatusAutostartStatusUnsupported;
}

int ColimaStatusSetAutostartEnabled(int enabled, char **errorMessage)
{
    if (errorMessage != NULL) {
        *errorMessage = NULL;
    }
    if (@available(macOS 13.0, *)) {
        SMAppService *service = SMAppService.mainAppService;
        int currentStatus = ColimaStatusMapAutostartStatus(service.status);

        if ((enabled && currentStatus == ColimaStatusAutostartStatusEnabled) ||
            (!enabled && currentStatus == ColimaStatusAutostartStatusDisabled)) {
            return currentStatus;
        }
        if (enabled && currentStatus == ColimaStatusAutostartStatusRequiresApproval) {
            return currentStatus;
        }

        NSError *error = nil;
        BOOL succeeded = enabled
            ? [service registerAndReturnError:&error]
            : [service unregisterAndReturnError:&error];
        int resultingStatus = ColimaStatusMapAutostartStatus(service.status);
        if (succeeded || resultingStatus == ColimaStatusAutostartStatusRequiresApproval) {
            return resultingStatus;
        }

        ColimaStatusSetErrorMessage(errorMessage, error);
        return ColimaStatusAutostartStatusError;
    }
    return ColimaStatusAutostartStatusUnsupported;
}

int ColimaStatusOpenAutostartSettings(void)
{
    if (@available(macOS 13.0, *)) {
        dispatch_async(dispatch_get_main_queue(), ^{
            [SMAppService openSystemSettingsLoginItems];
        });
        return 1;
    }
    return 0;
}
