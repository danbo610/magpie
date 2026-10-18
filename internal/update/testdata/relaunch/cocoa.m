#import <Cocoa/Cocoa.h>

void startCocoa(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited];
}

void quitCocoa(void) {
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, NSEC_PER_SEC / 3), dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
    [NSApp run];
}
