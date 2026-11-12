import SwiftUI

struct DriverLoadingView: View {
    let title: String
    var message: String = "Syncing manifest, route, and delivery assignments."

    @State private var animating = false

    var body: some View {
        VStack(spacing: LabTheme.s16) {
            ZStack {
                Circle()
                    .fill(LabTheme.fg.opacity(0.06))
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.04 : 0.96)
                ProgressView()
                    .controlSize(.regular)
                    .tint(LabTheme.fg)
            }

            VStack(spacing: LabTheme.s8) {
                Text(title.uppercased())
                    .font(.system(.subheadline, design: .monospaced, weight: .black))
                    .foregroundStyle(LabTheme.fg)
                Text(message)
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, minHeight: 180)
        .padding(LabTheme.s24)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title). \(message)")
        .onAppear {
            withAnimation(Anim.breathe) {
                animating = true
            }
        }
    }
}

struct DriverErrorView: View {
    let title: String
    let message: String
    let retry: () -> Void

    var body: some View {
        VStack(spacing: LabTheme.s16) {
            DriverStateCard(
                icon: "wifi.exclamationmark",
                title: title,
                message: message
            )
            Button("RETRY", action: retry)
                .font(.system(.subheadline, design: .monospaced, weight: .black))
                .frame(minWidth: 44, minHeight: 44)
                .buttonStyle(.pressable)
        }
        .frame(maxWidth: .infinity, minHeight: 180)
        .padding(LabTheme.s24)
    }
}

#Preview {
    ScrollView {
        VStack(spacing: 24) {
            DriverLoadingView(title: "Loading routes")
            DriverEmptyView(
                icon: "road.lanes",
                title: "No upcoming rides",
                message: "Pull to refresh or check back later.",
                actionTitle: "Refresh"
            ) {}
        }
    }
    .background(LabTheme.bg)
}
