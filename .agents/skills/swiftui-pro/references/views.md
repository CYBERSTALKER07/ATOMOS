# SwiftUI Views

- Strongly prefer to avoid breaking up view bodies using computed properties or methods that return `some View`, even if `@ViewBuilder` is used. Extract them into separate `View` structs instead, placing each into its own file.
- Flag `body` properties that are excessively long; they should be broken into extracted subviews.
- Button actions should be extracted from view bodies into separate methods, to avoid mixing layout and logic.
- Similarly, general business logic should not live inline in `task()`, `onAppear()` or elsewhere in `body`.
- Prefer to place view logic into view models or similar, so it can be tested. For more help with testing, suggest the [Swift Testing Pro agent skill](https://github.com/twostraws/swift-testing-agent-skill).
- Each type (struct, class, enum) should be in its own Swift file. Flag files containing multiple type definitions.
- Unless a full-screen editing experience is required, prefer using `TextField` with `axis: .vertical` to using `TextEditor`, because it allows placeholder text. If a specific minimum height is required for `TextField`, use something like `lineLimit(5...)`.
- If a button action can be provided directly as an `action` parameter, do so. For example: `Button("Label", systemImage: "plus", action: myAction)` is preferred over `Button("Label", systemImage: "plus") { action() }`.
- When rendering SwiftUI views to images, strongly prefer `ImageRenderer` over `UIGraphicsImageRenderer`.
- `#Preview` should be used for previews, not the legacy `PreviewProvider` protocol.
- When using `TabView(selection:)`, use a binding to a property that stores an enum rather than an integer or string. For example, `Tab("Home", systemImage: "house", value: .home)` is better than `Tab("Home", systemImage: "house", value: 0)`.
- Strongly prefer to avoid breaking up view bodies using computed properties or methods that return `some View`, even if `@ViewBuilder` is used. Extract them into separate `View` structs instead, placing each into its own file. (Yes, this is repeated, but it’s so important it needs to be mentioned twice.)


## Animating views

- Strongly prefer to use the `@Animatable` macro over creating `animatableData` manually – the macro automatically adds conformance to the `Animatable` protocol and creates the correct `animatableData` property. If some properties should not or cannot be animated (e.g. Booleans, integers, etc), mark them `@AnimatableIgnored`.
- Never use `animation(_ animation: Animation?)`; always provide a value to watch, such as `.animation(.bouncy, value: score)`.
- Chaining animations must be done using a `completion` closure passed to `withAnimation()`, rather than trying to execute multiple `withAnimation()` calls using delays.

For example:

```swift
Button("Animate Me") {
    withAnimation {
        scale = 2
    } completion: {
        withAnimation {
            scale = 1
        }
    }
}
```


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
