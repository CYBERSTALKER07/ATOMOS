import SwiftUI

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
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(LabTheme.fg)
            }
            .scaleEffect(reduceMotion ? 1 : (pulse ? 1.04 : 0.96))
            .onAppear {
                guard !reduceMotion else { return }
                withAnimation(Anim.breathe) {
                    pulse = true
                }
            }
            .accessibilityHidden(true)

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
            .accessibilityElement(children: .combine)

            if let actionTitle, let action {
                Button(actionTitle, action: action)
                    .frame(minWidth: 44, minHeight: 44)
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
