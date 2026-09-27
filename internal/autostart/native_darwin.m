#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#include <string.h>

// SMAppService requires macOS 13, which the deployment target guarantees, so
// none of this is guarded by @available. See APP_DEPLOYMENT_TARGET in
// Build/version.sh; Build/build.sh fails if the linked binary disagrees.

enum ColimaStatusAutostartStatus {
    ColimaStatusAutostartStatusError = -1,
    ColimaStatusAutostartStatusUnsupported = 0,
    ColimaStatusAutostartStatusDisabled = 1,
    ColimaStatusAutostartStatusEnabled = 2,
    ColimaStatusAutostartStatusRequiresApproval = 3,
    ColimaStatusAutostartStatusNotFound = 4,
};

static int ColimaStatusMapAutostartStatus(SMAppServiceStatus status)
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
    // The message is shown as a tooltip and also logged. localizedDescription
    // follows the system language, which is what the rest of the menu does too.
    NSString *message = error.localizedDescription;
    if (message == nil || message.length == 0) {
        message = @"Launch at login could not be changed";
    }
    *errorMessage = strdup(message.UTF8String);
}

int ColimaStatusAutostartStatus(void)
{
    return ColimaStatusMapAutostartStatus(SMAppService.mainAppService.status);
}

int ColimaStatusSetAutostartEnabled(int enabled, char **errorMessage)
{
    if (errorMessage != NULL) {
        *errorMessage = NULL;
    }

    SMAppService *service = SMAppService.mainAppService;
    int currentStatus = ColimaStatusMapAutostartStatus(service.status);

    if ((enabled && currentStatus == ColimaStatusAutostartStatusEnabled) ||
        (!enabled && currentStatus == ColimaStatusAutostartStatusDisabled)) {
        return currentStatus;
    }
    // Approval is the user's to give in System Settings; registering again
    // would not move it forward.
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

int ColimaStatusOpenAutostartSettings(void)
{
    dispatch_async(dispatch_get_main_queue(), ^{
        [SMAppService openSystemSettingsLoginItems];
    });
    return 1;
}
