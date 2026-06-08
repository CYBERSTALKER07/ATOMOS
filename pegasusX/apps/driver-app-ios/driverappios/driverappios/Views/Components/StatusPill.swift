//
//  StatusPill.swift
//  driverappios
//

import SwiftUI

struct StatusPill: View {
    let label: String
    let color: Color
    @State private var appeared = false

    var body: some View {
        Text(label)
            .font(.system(size: 10, weight: .bold, design: .monospaced))
            .foregroundStyle(color)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(color.opacity(0.12), in: Capsule())
            .overlay(Capsule().stroke(color.opacity(0.2), lineWidth: 0.5))
            .scaleEffect(appeared ? 1 : 0.8)
            .opacity(appeared ? 1 : 0)
            .onAppear {
                withAnimation(Anim.bouncy) { appeared = true }
            }
    }
}

#Preview {
    VStack(spacing: 12) {
        StatusPill(label: "EN_ROUTE", color: .primary)
        StatusPill(label: "DELIVERED", color: LabTheme.success)
        StatusPill(label: "REJECTED", color: LabTheme.destructive)
        StatusPill(label: "OFFLINE", color: LabTheme.warning)
    }
    .padding()
}
