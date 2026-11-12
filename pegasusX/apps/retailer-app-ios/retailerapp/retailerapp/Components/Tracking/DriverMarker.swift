import SwiftUI

struct DriverMarker: View {
    let isGreen: Bool

    var body: some View {
        ZStack {
            Circle()
                .fill(isGreen ? AppTheme.success : AppTheme.accent)
                .frame(width: 32, height: 32)
            Image(systemName: "box.truck.fill")
                .font(.system(.subheadline, weight: .bold))
                .foregroundStyle(.white)
        }
        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
        .accessibilityElement(children: .combine)
        .accessibilityLabel(isGreen ? "Driver, approaching" : "Driver")
    }
}
