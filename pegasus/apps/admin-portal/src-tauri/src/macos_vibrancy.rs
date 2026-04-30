#[cfg(target_os = "macos")]
use tauri::WebviewWindow;

/// Apply macOS vibrancy (translucent sidebar effect) to the main window.
#[cfg(target_os = "macos")]
pub fn apply_vibrancy(window: &WebviewWindow) {
    use cocoa::appkit::{
        NSVisualEffectMaterial, NSVisualEffectState, NSVisualEffectView,
        NSWindow, NSWindowStyleMask,
    };
    use cocoa::base::{id, nil};
    use cocoa::foundation::NSRect;
    use objc::runtime::Object;
    use objc::{msg_send, sel, sel_impl};

    unsafe {
        let ns_window: id = window.ns_window().expect("Failed to get NSWindow") as id;

        // Enable titlebar transparency
        let mask = ns_window.styleMask();
        ns_window.setStyleMask_(
            mask | NSWindowStyleMask::NSFullSizeContentViewWindowMask,
        );
        ns_window.setTitlebarAppearsTransparent_(true);

        // Set window background to clear
        let clear_color: id = msg_send![objc::class!(NSColor), clearColor];
        ns_window.setBackgroundColor_(clear_color);

        // Create vibrancy effect view
        let content_view: id = ns_window.contentView();
        let bounds: NSRect = msg_send![content_view, bounds];

        let visual_effect: id = NSVisualEffectView::alloc(nil).initWithFrame_(bounds);
        visual_effect.setMaterial_(NSVisualEffectMaterial::UnderWindowBackground);
        visual_effect.setState_(NSVisualEffectState::FollowsWindowActiveState);
        visual_effect.setBlendingMode_(
            cocoa::appkit::NSVisualEffectBlendingMode::BehindWindow,
        );

        // Set autoresizing mask so vibrancy fills window
        let autoresize: u64 = 18; // NSViewWidthSizable | NSViewHeightSizable
        let _: () = msg_send![visual_effect, setAutoresizingMask: autoresize];

        // Insert vibrancy view behind content
        let _: () = msg_send![content_view, addSubview: visual_effect positioned: 1_i64 relativeTo: nil];
    }

    log::info!("macOS vibrancy applied.");
}
