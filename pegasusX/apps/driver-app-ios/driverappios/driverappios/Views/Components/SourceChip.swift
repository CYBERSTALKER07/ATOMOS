import SwiftUI

struct SourceChip: View {
    let source: String

    var body: some View {
        Text(source.uppercased())
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .overlay(Capsule().strokeBorder(.secondary, lineWidth: 1))
            .accessibilityIdentifier("gs-u-source-chip")
            .accessibilityLabel("Source: \(source)")
    }
}

struct CommandStatusJump: Hashable, Identifiable {
    let id: UUID
    let status: String
    var supplierId: String?

    init(status: String, supplierId: String? = nil) {
        self.id = UUID()
        self.status = canonicalizeOrderStatus(status)
        self.supplierId = supplierId
    }
}
