# Accessibility

- Respect the user’s accessibility settings for fonts, colors, animations, and more.
- Do not force specific font sizes. Prefer Dynamic Type (`.font(.body)`, `.font(.headline)`, etc.).
- If you *need* a custom font size, use `@ScaledMetric` when targeting iOS 18 and earlier. When targeting iOS 26 or later, `.font(.body.scaled(by:))` is also available to get font size adjustment.
- Flag instances where images have unclear or unhelpful VoiceOver readings, e.g. `Image(.newBanner2026)`. If they are decorative, suggest using `Image(decorative:)` or `accessibilityHidden()`, otherwise attach an `accessibilityLabel()`.
- If the user has “Reduce Motion” enabled, replace large, motion-based animations with opacity instead.
- If buttons have complex or frequently changing labels, recommend using `accessibilityInputLabels()` to provide better Voice Control commands. For example, if a button had a live-updating share price for Apple such as “AAPL $271.68”, adding an input label for “Apple” would be a big improvement.
- Buttons with image labels must always include text, even if the text is invisible: `Button("Label", systemImage: "plus", action: myAction)`. Flag icon-only buttons that lack a text label as being bad for VoiceOver.
- If color is an important differentiator in the user interface, make sure to respect the environment’s `.accessibilityDifferentiateWithoutColor` setting by showing some kind of variation beyond just color – icons, patterns, strokes, etc.
- The same is true of `Menu`: using `Menu("Options", systemImage: "ellipsis.circle") { }` is much better than just using an image.
- Never use `onTapGesture()` unless you specifically need tap location or tap count. All other tappable elements should be a `Button`.
- If `onTapGesture()` must be used, make sure to add `.accessibilityAddTraits(.isButton)` or similar so it can be read by VoiceOver correctly.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
