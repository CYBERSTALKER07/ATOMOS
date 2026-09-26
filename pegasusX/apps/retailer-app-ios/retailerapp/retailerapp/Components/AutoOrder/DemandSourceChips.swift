import SwiftUI

/// Enterprise demand-source chips for reorder / sell-through surfaces.
/// STORE_POS = floor sales velocity; WHOLESALE_HISTORY = B2B demand sensing.
struct DemandSourceChips: View {
    let sources: [String]?

    private var list: [String] {
        if let sources, !sources.isEmpty { return sources }
        return ["WHOLESALE_HISTORY"]
    }

    var body: some View {
        HStack(spacing: 6) {
            ForEach(list, id: \.self) { code in
                Text(label(for: code).uppercased())
                    .font(.system(size: 10, weight: .semibold, design: .rounded))
                    .tracking(0.4)
                    .foregroundStyle(isPOS(code) ? Color(red: 0.08, green: 0.50, blue: 0.24) : AppTheme.textSecondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(
                        Capsule().fill(
                            isPOS(code)
                                ? Color(red: 0.09, green: 0.64, blue: 0.29).opacity(0.18)
                                : AppTheme.surfaceElevated
                        )
                    )
                    .accessibilityLabel("Demand source: \(label(for: code))")
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Demand sources: \(list.map(label).joined(separator: ", "))")
    }

    private func isPOS(_ code: String) -> Bool {
        code.uppercased() == "STORE_POS"
    }

    private func label(for code: String) -> String {
        switch code.uppercased() {
        case "STORE_POS": return "Store POS"
        case "WHOLESALE_HISTORY": return "Wholesale"
        default: return code
        }
    }
}
