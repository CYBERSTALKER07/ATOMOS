## 2025-05-14 - [A11y: Improving Interactive Context and Feedback]
**Learning:** Interactive elements like custom tabs and status messages are often technically "invisible" to screen readers unless proper ARIA roles (tablist, tab, status) are applied. Non-semantic elements (divs) used as buttons lack focusability and clear purpose for keyboard users.
**Action:** Always use semantic `<button>` elements for interactive indicators, apply `role="status"` to dynamic feedback messages, and implement the full WAI-ARIA Tabs pattern for content switchers to ensure a predictable experience for all users.

## 2025-05-14 - [A11y: Implementing Global Skip Links and Semantic Landmarks]
**Learning:** For large applications with complex navigation, a 'Skip to content' link is essential for keyboard accessibility. Centralizing the `<main>` landmark in the root layout is more maintainable than individual page landmarks, but requires ensuring that sub-layouts don't inadvertently include navigation within the landmark.
**Action:** Implement 'Skip to content' links in the root `layout.tsx` targeting a centralized `<main id="main-content">` that wraps `{children}`, and ensure high z-index (10001+) to stay above splash screens or other overlays.
