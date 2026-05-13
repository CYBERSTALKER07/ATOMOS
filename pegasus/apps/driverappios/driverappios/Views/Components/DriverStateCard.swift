import SwiftUI

struct DriverLoadingStateCard: View {
    let title: String
    let message: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.s16) {
            HStack(spacing: 12) {
                ProgressView()
                    .controlSize(.large)
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.headline)
                        .foregroundStyle(LabTheme.fg)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(LabTheme.fgSecondary)
                }
            }

            VStack(spacing: 10) {
                DriverStateSkeletonBar(width: nil)
                DriverStateSkeletonBar(width: nil)
                DriverStateSkeletonBar(width: 180)
            }
        }
        .padding(24)
        .frame(maxWidth: 420)
        .labCard()
        .staggeredAppear(index: 0)
    }
}

struct DriverStateCard: View {
    let icon: String
    let title: String
    let message: String
    var actionTitle: String? = nil
    var action: (() -> Void)? = nil

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var pulse = false

    var body: some View {
        VStack(spacing: LabTheme.s16) {
            ZStack {
                Circle()
                    .fill(LabTheme.separator.opacity(0.12))
                    .frame(width: 64, height: 64)
                Image(systemName: icon)
                    .font(.system(size: 24, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
            }
            .scaleEffect(reduceMotion ? 1 : (pulse ? 1.04 : 0.96))
            .onAppear {
                guard !reduceMotion else { return }
                withAnimation(Anim.breathe) {
                    pulse = true
                }
            }

            VStack(spacing: 6) {
                Text(title)
                    .font(.headline)
                    .foregroundStyle(LabTheme.fg)
                    .multilineTextAlignment(.center)
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(LabTheme.fgSecondary)
                    .multilineTextAlignment(.center)
            }

            if let actionTitle, let action {
                Button(actionTitle, action: action)
                    .buttonStyle(.borderedProminent)
                    .buttonStyle(.pressable)
            }
        }
        .padding(24)
        .frame(maxWidth: 420)
        .labCard()
        .staggeredAppear(index: 0)
    }
}

private struct DriverStateSkeletonBar: View {
    let width: CGFloat?

    var body: some View {
        RoundedRectangle(cornerRadius: 10)
            .fill(LabTheme.separator.opacity(0.14))
            .frame(maxWidth: width ?? .infinity)
            .frame(height: 14)
            .shimmer()
    }
}

#Preview {
    VStack(spacing: 20) {
        DriverLoadingStateCard(
            title: "Loading notifications",
            message: "Checking route, payment, and dispatch updates."
        )
        DriverStateCard(
            icon: "bell.slash",
            title: "No notifications yet",
            message: "Dispatch updates will appear here."
        )
        DriverStateCard(
            icon: "wifi.exclamationmark",
            title: "Couldn't load notifications",
            message: "Check your connection and try again.",
            actionTitle: "Retry"
        ) {}
    }
    .padding()
    .background(LabTheme.bg)
}