import SwiftUI

enum FactoryStateKind {
    case empty
    case noResults
    case error
    case offline
    case restricted
    case authFailure
}

enum FactoryRuntimeTone {
    case live
    case refreshing
    case warning
    case offline
}

struct FactoryLoadingState: View {
    let title: String
    let message: String

    @State private var animating = false

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(LabTheme.tertiaryBackground)
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.04 : 0.96)

                ProgressView()
                    .controlSize(.regular)
            }

            VStack(spacing: LabTheme.spacingSM) {
                Text(title)
                    .font(.title3.bold())
                Text(message)
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(LabTheme.spacingXL)
        .background(LabTheme.background)
        .onAppear {
            withAnimation(Anim.smooth.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }
}

struct FactoryStateView: View {
    let kind: FactoryStateKind
    let headline: String
    let message: String
    var actionTitle: String? = nil
    var action: (() -> Void)? = nil

    @State private var animating = false

    var body: some View {
        let palette = paletteFor(kind)

        VStack(spacing: LabTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(palette.fill)
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.03 : 0.97)

                Image(systemName: palette.icon)
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(palette.tint)
            }

            VStack(spacing: LabTheme.spacingSM) {
                Text(headline)
                    .font(.title3.bold())
                Text(message)
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }

            if let actionTitle, let action {
                Button(actionTitle, action: action)
                    .buttonStyle(.borderedProminent)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(LabTheme.spacingXL)
        .background(LabTheme.background)
        .onAppear {
            withAnimation(Anim.smooth.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }

    private func paletteFor(_ kind: FactoryStateKind) -> (icon: String, fill: Color, tint: Color) {
        switch kind {
        case .empty:
            return ("tray", LabTheme.tertiaryBackground, LabTheme.secondaryLabel)
        case .noResults:
            return ("magnifyingglass", LabTheme.tertiaryBackground, LabTheme.secondaryLabel)
        case .error:
            return ("exclamationmark.triangle", LabTheme.destructive.opacity(0.16), LabTheme.destructive)
        case .offline:
            return ("wifi.slash", LabTheme.tertiaryBackground, LabTheme.secondaryLabel)
        case .restricted:
            return ("lock.fill", LabTheme.fill, LabTheme.label)
        case .authFailure:
            return ("key.fill", LabTheme.destructive.opacity(0.16), LabTheme.destructive)
        }
    }
}

/// Inline loading pane for scroll contexts (lighter than full-screen `FactoryLoadingState`).
struct FactoryLoadingView: View {
    let title: String
    var message: String = "Fetching the latest factory data."

    @State private var animating = false

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(LabTheme.tertiaryBackground)
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.04 : 0.96)
                ProgressView()
                    .controlSize(.regular)
            }

            VStack(spacing: LabTheme.spacingSM) {
                Text(title)
                    .font(.title3.bold())
                Text(message)
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(LabTheme.spacingXL)
        .onAppear {
            withAnimation(Anim.smooth.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
    }
}

struct FactoryErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("Unable to load", systemImage: "exclamationmark.triangle")
        } description: {
            Text(message)
        } actions: {
            Button("Retry", action: retry)
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
    }
}

struct FactoryRuntimeBanner: View {
    let tone: FactoryRuntimeTone
    let message: String

    var body: some View {
        let palette = paletteFor(tone)

        HStack(spacing: LabTheme.spacingSM) {
            Image(systemName: palette.icon)
                .font(.footnote.weight(.semibold))
            Text(message)
                .font(.footnote.weight(.medium))
        }
        .foregroundStyle(palette.tint)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, LabTheme.spacingMD)
        .padding(.vertical, LabTheme.spacingSM)
        .background(palette.fill, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }

    private func paletteFor(_ tone: FactoryRuntimeTone) -> (icon: String, fill: Color, tint: Color) {
        switch tone {
        case .live:
            return ("arrow.triangle.2.circlepath", LabTheme.tertiaryBackground, LabTheme.secondaryLabel)
        case .refreshing:
            return ("arrow.triangle.2.circlepath", LabTheme.fill, LabTheme.label)
        case .warning:
            return ("exclamationmark.triangle", LabTheme.warning.opacity(0.16), LabTheme.warning)
        case .offline:
            return ("wifi.slash", LabTheme.tertiaryBackground, LabTheme.secondaryLabel)
        }
    }
}