import SwiftUI

struct OnlineDot: View {
    let online: Bool
    let queued: Int

    @State private var pulseScale: CGFloat = 1

    var body: some View {
        HStack(spacing: TermTheme.s8) {
            ZStack {
                if online {
                    Circle()
                        .fill(TermTheme.live.opacity(0.35))
                        .frame(width: 14, height: 14)
                        .scaleEffect(pulseScale)
                        .animation(
                            .easeInOut(duration: 1.2).repeatForever(autoreverses: true),
                            value: pulseScale
                        )
                }
                Circle()
                    .fill(online ? TermTheme.live : TermTheme.alert)
                    .frame(width: 8, height: 8)
            }
            Text(statusLabel)
                .font(.system(size: 12, weight: .bold, design: .monospaced))
                .foregroundStyle(TermTheme.secondary)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(accessibilityLabel)
        .onAppear { if online { pulseScale = 1.6 } }
        .onChange(of: online) { _, isOnline in
            pulseScale = isOnline ? 1.6 : 1
        }
    }

    private var statusLabel: String {
        if online { return "LIVE" }
        if queued > 0 { return "OFFLINE · \(queued) QUEUED" }
        return "OFFLINE"
    }

    private var accessibilityLabel: String {
        if online { return "Connected live" }
        if queued > 0 { return "Offline, \(queued) actions queued" }
        return "Offline"
    }
}
