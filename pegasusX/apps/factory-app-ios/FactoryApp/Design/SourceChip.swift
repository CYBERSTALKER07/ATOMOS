import SwiftUI

struct SourceChip: View {
    let source: String

    var body: some View {
        Text(source.uppercased())
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .overlay(Capsule().strokeBorder(.secondary, lineWidth: 1))
            .accessibilityLabel("Data source: \(source)")
            .accessibilityIdentifier("gs-u-source-chip")
    }
}
