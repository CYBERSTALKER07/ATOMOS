//
//  GPSErrorBanner.swift
//  driverappios
//

import SwiftUI

/// Tactical banner when GPS is unavailable or permission denied.
struct GPSErrorBanner: View {
    let message: String

    var body: some View {
        HStack(spacing: LabTheme.s12) {
            Image(systemName: "location.slash.fill")
                .font(.system(.subheadline, weight: .bold))

            VStack(alignment: .leading, spacing: 2) {
                Text("GPS_UNAVAILABLE")
                    .font(.system(.caption2, design: .monospaced, weight: .black))
                    .tracking(1.2)
                Text(message)
                    .font(.system(.caption, weight: .medium))
                    .lineLimit(2)
            }

            Spacer(minLength: 0)
        }
        .foregroundStyle(.white)
        .padding(.horizontal, LabTheme.s16)
        .padding(.vertical, LabTheme.s12)
        .background(LabTheme.destructive.gradient, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous))
        .padding(.horizontal, LabTheme.s16)
        .transition(.move(edge: .top).combined(with: .opacity))
    }
}

#Preview {
    VStack {
        GPSErrorBanner(message: "Location permission denied. Enable in Settings.")
        Spacer()
    }
    .background(LabTheme.bg)
}
