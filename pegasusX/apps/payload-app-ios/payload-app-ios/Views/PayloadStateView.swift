import SwiftUI

enum PayloadStateVariant {
    case sync
    case truck
    case manifest
    case dispatch
    case notifications
    case warning
    case success
}

enum PayloadStateTone {
    case `default`
    case warning
    case success
}

struct PayloadStateView: View {
    let variant: PayloadStateVariant
    let title: String
    let message: String?
    var detail: String? = nil
    var compact = false
    var tone: PayloadStateTone = .default

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var pulse = false
    @State private var drift = false
    @State private var spin = false

    private var accent: Color {
        switch tone {
        case .default:
            return TermTheme.accent
        case .warning:
            return TermTheme.warn
        case .success:
            return TermTheme.live
        }
    }

    private var shouldRotate: Bool {
        variant == .sync || variant == .dispatch
    }

    private var illustrationSize: CGFloat {
        compact ? 96 : 136
    }

    var body: some View {
        VStack(spacing: compact ? 12 : 18) {
            ZStack {
                Circle()
                    .fill(accent.opacity(0.10))
                    .frame(width: illustrationSize * 0.72, height: illustrationSize * 0.72)
                    .scaleEffect(reduceMotion ? 1 : (pulse ? 1.08 : 0.92))
                    .opacity(reduceMotion ? 0.18 : (pulse ? 0.10 : 0.24))

                illustration
                    .frame(width: illustrationSize, height: illustrationSize)
                    .scaleEffect(reduceMotion ? 1 : (pulse ? 1.03 : 0.985))
                    .offset(y: reduceMotion ? 0 : (drift ? -4 : 3))
                    .rotationEffect(shouldRotate && !reduceMotion ? .degrees(spin ? 360 : 0) : .zero)
            }

            Text(title)
                .font(.system(size: compact ? 16 : 24, weight: .black, design: .monospaced))
                .foregroundStyle(TermTheme.accent)
                .multilineTextAlignment(.center)

            if let message, !message.isEmpty {
                Text(message)
                    .font(.system(size: compact ? 12 : 14, weight: .medium, design: .monospaced))
                    .foregroundStyle(TermTheme.secondary)
                    .multilineTextAlignment(.center)
            }

            if let detail, !detail.isEmpty {
                Text(detail)
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundStyle(TermTheme.tertiary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: compact ? 280 : 360)
        .onAppear {
            guard !reduceMotion else { return }
            pulse = true
            drift = true
            if shouldRotate {
                spin = true
            }
        }
        .animation(.easeInOut(duration: compact ? 1.0 : 1.2).repeatForever(autoreverses: true), value: pulse)
        .animation(.easeInOut(duration: compact ? 1.4 : 1.8).repeatForever(autoreverses: true), value: drift)
        .animation(.linear(duration: variant == .sync ? 3.2 : 2.8).repeatForever(autoreverses: false), value: spin)
    }

    @ViewBuilder
    private var illustration: some View {
        switch variant {
        case .sync:
            ZStack {
                Circle()
                    .stroke(TermTheme.tertiary.opacity(0.55), style: StrokeStyle(lineWidth: 2, dash: [5, 7]))
                    .frame(width: illustrationSize * 0.7, height: illustrationSize * 0.7)
                RoundedRectangle(cornerRadius: compact ? 18 : 24, style: .continuous)
                    .fill(TermTheme.card)
                    .frame(width: illustrationSize * 0.42, height: illustrationSize * 0.62)
                    .overlay {
                        RoundedRectangle(cornerRadius: compact ? 18 : 24, style: .continuous)
                            .stroke(TermTheme.secondary.opacity(0.6), lineWidth: 2)
                    }
                RoundedRectangle(cornerRadius: compact ? 10 : 12, style: .continuous)
                    .fill(accent.opacity(0.14))
                    .frame(width: illustrationSize * 0.22, height: illustrationSize * 0.24)
                    .offset(y: -8)
                Image(systemName: "arrow.triangle.2.circlepath")
                    .font(.system(size: compact ? 24 : 32, weight: .bold))
                    .foregroundStyle(accent)
                    .offset(x: illustrationSize * 0.2, y: -illustrationSize * 0.22)
            }
        case .truck:
            ZStack {
                RoundedRectangle(cornerRadius: compact ? 10 : 12, style: .continuous)
                    .fill(accent.opacity(0.14))
                    .frame(width: illustrationSize * 0.32, height: illustrationSize * 0.14)
                    .offset(x: -10, y: -26)
                Image(systemName: "truck.box.fill")
                    .font(.system(size: compact ? 40 : 54, weight: .bold))
                    .foregroundStyle(accent)
                HStack(spacing: illustrationSize * 0.14) {
                    Circle().fill(TermTheme.card).frame(width: compact ? 12 : 14, height: compact ? 12 : 14)
                    Circle().fill(TermTheme.card).frame(width: compact ? 12 : 14, height: compact ? 12 : 14)
                }
                .offset(y: illustrationSize * 0.22)
            }
        case .manifest:
            ZStack {
                RoundedRectangle(cornerRadius: compact ? 16 : 22, style: .continuous)
                    .fill(TermTheme.card)
                    .frame(width: illustrationSize * 0.56, height: illustrationSize * 0.74)
                    .overlay {
                        RoundedRectangle(cornerRadius: compact ? 16 : 22, style: .continuous)
                            .stroke(TermTheme.secondary.opacity(0.55), lineWidth: 2)
                    }
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .fill(accent.opacity(0.16))
                    .frame(width: illustrationSize * 0.24, height: illustrationSize * 0.09)
                    .offset(y: -illustrationSize * 0.26)
                Image(systemName: "checklist")
                    .font(.system(size: compact ? 28 : 38, weight: .bold))
                    .foregroundStyle(accent)
            }
        case .dispatch:
            ZStack {
                Path { path in
                    path.move(to: CGPoint(x: illustrationSize * 0.26, y: illustrationSize * 0.7))
                    path.addCurve(
                        to: CGPoint(x: illustrationSize * 0.72, y: illustrationSize * 0.36),
                        control1: CGPoint(x: illustrationSize * 0.42, y: illustrationSize * 0.62),
                        control2: CGPoint(x: illustrationSize * 0.58, y: illustrationSize * 0.46)
                    )
                }
                .stroke(accent, style: StrokeStyle(lineWidth: compact ? 4 : 5, lineCap: .round))
                Circle()
                    .fill(TermTheme.card)
                    .frame(width: compact ? 22 : 26, height: compact ? 22 : 26)
                    .overlay(Circle().stroke(TermTheme.secondary.opacity(0.6), lineWidth: 2))
                    .offset(x: -illustrationSize * 0.22, y: illustrationSize * 0.18)
                Circle()
                    .fill(TermTheme.card)
                    .frame(width: compact ? 22 : 26, height: compact ? 22 : 26)
                    .overlay(Circle().stroke(TermTheme.secondary.opacity(0.6), lineWidth: 2))
                    .offset(x: illustrationSize * 0.22, y: -illustrationSize * 0.18)
                Image(systemName: "arrow.triangle.swap")
                    .font(.system(size: compact ? 24 : 30, weight: .bold))
                    .foregroundStyle(accent)
            }
        case .notifications:
            ZStack {
                Image(systemName: "bell.fill")
                    .font(.system(size: compact ? 42 : 54, weight: .bold))
                    .foregroundStyle(accent)
                Circle()
                    .fill(accent)
                    .frame(width: compact ? 14 : 18, height: compact ? 14 : 18)
                    .offset(x: illustrationSize * 0.16, y: -illustrationSize * 0.2)
                Circle()
                    .fill(TermTheme.card)
                    .frame(width: compact ? 6 : 8, height: compact ? 6 : 8)
                    .offset(x: illustrationSize * 0.16, y: -illustrationSize * 0.2)
            }
        case .warning:
            ZStack {
                TriangleShape()
                    .fill(accent.opacity(0.16))
                    .frame(width: illustrationSize * 0.6, height: illustrationSize * 0.52)
                Image(systemName: "exclamationmark.triangle.fill")
                    .font(.system(size: compact ? 38 : 52, weight: .bold))
                    .foregroundStyle(accent)
            }
        case .success:
            ZStack {
                Circle()
                    .fill(accent.opacity(0.12))
                    .frame(width: illustrationSize * 0.62, height: illustrationSize * 0.62)
                Image(systemName: "checkmark.seal.fill")
                    .font(.system(size: compact ? 40 : 54, weight: .bold))
                    .foregroundStyle(accent)
            }
        }
    }
}

private struct TriangleShape: Shape {
    func path(in rect: CGRect) -> Path {
        Path { path in
            path.move(to: CGPoint(x: rect.midX, y: rect.minY))
            path.addLine(to: CGPoint(x: rect.maxX, y: rect.maxY))
            path.addLine(to: CGPoint(x: rect.minX, y: rect.maxY))
            path.closeSubpath()
        }
    }
}

#Preview {
    PayloadStateView(
        variant: .truck,
        title: "No vehicles",
        message: "Pull to refresh once dispatch assigns trucks.",
        compact: false,
        tone: .warning
    )
    .padding()
    .background(TermTheme.bg)
}