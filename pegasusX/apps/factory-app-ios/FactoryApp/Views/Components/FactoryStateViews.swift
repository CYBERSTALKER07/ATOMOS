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

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var animating = false

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(LabTheme.tertiaryBackground)
                    .frame(width: 72, height: 72)
                    .scaleEffect(reduceMotion ? 1.0 : (animating ? 1.04 : 0.96))

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
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title). \(message)")
        .onAppear {
            guard !reduceMotion else { return }
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

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var animating = false

    var body: some View {
        let palette = paletteFor(kind)

        VStack(spacing: LabTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(palette.fill)
                    .frame(width: 72, height: 72)
                    .scaleEffect(reduceMotion ? 1.0 : (animating ? 1.03 : 0.97))

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
                Button(actionTitle) {
                    PegasusHaptics.selection()
                    action()
                }
                .buttonStyle(.borderedProminent)
                .frame(minWidth: 44, minHeight: 44)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(LabTheme.spacingXL)
        .background(LabTheme.background)
        .accessibilityElement(children: .contain)
        .onAppear {
            guard !reduceMotion else { return }
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
