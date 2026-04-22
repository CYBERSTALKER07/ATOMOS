//
//  TelemetryBadge.swift
//  driverappios
//

import SwiftUI

struct TelemetryBadge: View {
    let isLive: Bool
    @State private var pulse = false

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(isLive ? LabTheme.live : LabTheme.offline)
                .frame(width: 7, height: 7)
                .scaleEffect(pulse && isLive ? 1.4 : 1)
                .opacity(pulse && isLive ? 0.6 : 1)

            Text(isLive ? "LIVE" : "OFFLINE")
                .font(.system(size: 10, weight: .heavy, design: .monospaced))
                .foregroundStyle(isLive ? LabTheme.live : LabTheme.offline)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(.ultraThinMaterial, in: Capsule())
        .overlay(Capsule().stroke(Color.white.opacity(0.08), lineWidth: 0.5))
        .onAppear {
            if isLive {
                withAnimation(Anim.breathe) { pulse = true }
            }
        }
        .onChange(of: isLive) { _, live in
            if live {
                withAnimation(Anim.breathe) { pulse = true }
            } else {
                pulse = false
            }
        }
    }
}

#Preview {
    VStack(spacing: 12) {
        TelemetryBadge(isLive: true)
        TelemetryBadge(isLive: false)
    }
    .padding()
    .background(Color.gray)
}
