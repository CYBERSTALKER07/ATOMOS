# Design

## Creating a uniform design in this app

Prefer to place standard fonts, sizes, colors, stack spacing, padding, rounding, animation timings, and more into a shared enum of constants, so they can be used by all views. This allows the app’s design to feel uniform and consistent, and be adjusted easily.


## Requirements for flexible, accessible design

- Never use `UIScreen.main.bounds` to read available space; prefer alternatives such as `containerRelativeFrame()`, or `visualEffect()` as appropriate, or (if there is no alternative) `GeometryReader`.
- Prefer to avoid fixed frames for views unless content can fit neatly inside; this can cause problems across different device sizes, different Dynamic Type settings, and more. Giving frames some flexibility is usually preferred.
- Apple’s minimum acceptable tap area for interactions on iOS is 44x44. Ensure this is strictly enforced.


## Standard system styling

- Strongly prefer to use `ContentUnavailableView` when data is missing or empty, rather than designing something custom.
- When using `searchable()`, you can show empty results using `ContentUnavailableView.search` and it will include the search term they used automatically – there’s no need to use `ContentUnavailableView.search(text: searchText)` or similar.
- If you need an icon and some text placed horizontally side by side, prefer `Label` over `HStack`.
- Prefer system hierarchical styles (e.g. secondary/tertiary) over manual opacity when possible, so the system can adapt to the correct context automatically.
- When using `Form`, wrap controls such as `Slider` in `LabeledContent` so the title and control are laid out correctly.
- When using `RoundedRectangle`, the default rounding style is `.continuous` – there is no need to specify it explicitly.


## Ensuring designs work for everyone

- Use `bold()` instead of `fontWeight(.bold)`, because using `bold()` allows the system to choose the correct weight for the current context.
- Only use `fontWeight()` for weights other than bold when there's an important reason - scattering around `fontWeight(.medium)` or `fontWeight(.semibold)` is counterproductive.
- Avoid hard-coded values for padding and stack spacing unless specifically requested.
- Avoid UIKit colors (`UIColor`) in SwiftUI code; use SwiftUI `Color` or asset catalog colors.
- The font size `.caption2` is extremely small, and is generally best avoided. Even the font size `.caption` is on the small side, and should be used carefully.


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
