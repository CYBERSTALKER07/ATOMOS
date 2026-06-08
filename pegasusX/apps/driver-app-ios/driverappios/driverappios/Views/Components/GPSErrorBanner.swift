//
//  GPSErrorBanner.swift
//  driverappios
//

import SwiftUI

/// Red banner displayed at the top when GPS is unavailable or permission denied.
struct GPSErrorBanner: View {
    let message: String

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: "location.slash.fill")
                .font(.subheadline.weight(.semibold))

            Text(message)
                .font(.caption.weight(.medium))
                .lineLimit(2)

            Spacer()
        }
        .foregroundStyle(.white)
        .padding(.horizontal, LabTheme.s16)
        .padding(.vertical, 10)
        .background(Color.black.gradient)
        .transition(.move(edge: .top).combined(with: .opacity))
    }
}

#Preview {
    VStack {
        GPSErrorBanner(message: "Location permission denied. Enable in Settings.")
        Spacer()
    }
}
