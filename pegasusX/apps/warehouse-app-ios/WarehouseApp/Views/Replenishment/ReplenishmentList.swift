import SwiftUI

struct ReplenishmentList: View {
    let insights: [ReplenishmentInsight]
    let actingId: String?
    let onApprove: (String) -> Void
    let onDismiss: (String) -> Void

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(insights) { insight in
                VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                    HStack(spacing: LabTheme.spacingXS) {
                        Text(insight.productName)
                            .font(.headline)
                        if insight.reasonCode == "PREDICTIVE_PUSH" {
                            Text("AI PUSH")
                                .font(.system(size: 10, weight: .bold))
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.primary)
                                .foregroundColor(Color(UIColor.systemBackground))
                                .clipShape(Capsule())
                        }
                    }
                    HStack(spacing: LabTheme.spacingSM) {
                        WarehouseStatusBadge(text: insight.urgency, tint: urgencyTint(insight.urgency))
                        WarehouseStatusBadge(text: insight.status)
                    }
                    Text("Stock \(insight.currentStock) · Reorder \(insight.reorderQuantity)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("Days until stockout: \(insight.daysUntilStockout)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if let why = demandWhyText(insight) {
                        Text(why)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if insight.status.uppercased() == "OPEN" {
                        HStack {
                            Button("Approve") {
                                onApprove(insight.id)
                            }
                            .disabled(actingId == insight.id)
                            Button("Dismiss", role: .destructive) {
                                onDismiss(insight.id)
                            }
                            .disabled(actingId == insight.id)
                        }
                        .buttonStyle(.bordered)
                    }
                }
                .padding(.vertical, LabTheme.spacingXS)
            }
        }
    }

    private func urgencyTint(_ urgency: String) -> Color {
        switch urgency.uppercased() {
        case "PROACTIVE": return .primary
        case "CRITICAL": return LabTheme.destructive
        case "URGENT": return LabTheme.warning
        default: return LabTheme.secondaryLabel
        }
    }

    private func demandWhyText(_ insight: ReplenishmentInsight) -> String? {
        if let breakdown = insight.demandBreakdown, !breakdown.isEmpty {
            if let blocked = string(from: breakdown["blocked_reason"]), !blocked.isEmpty {
                return blocked == "insufficient_history"
                    ? "Insufficient history — forecast blocked"
                    : blocked.replacingOccurrences(of: "_", with: " ")
            }
            var parts: [String] = []
            if let burn = number(from: breakdown["burn_rate_7d"]) ?? number(from: breakdown["burn_rate"]) {
                parts.append(String(format: "Burn %.1f/d", burn))
            }
            if let cover = number(from: breakdown["days_cover"]) {
                parts.append(String(format: "%.1fd cover", cover))
            }
            if breakdown["mei_network"] != nil {
                parts.append("MEIO network transfer")
            }
            if !parts.isEmpty { return parts.joined(separator: " · ") }
        }
        if let code = insight.reasonCode, !code.isEmpty {
            return code.replacingOccurrences(of: "_", with: " ")
        }
        return nil
    }

    private func number(from codable: AnyCodable?) -> Double? {
        guard let codable else { return nil }
        if let d = codable.value as? Double { return d }
        if let i = codable.value as? Int { return Double(i) }
        if let s = codable.value as? String, let d = Double(s) { return d }
        return nil
    }

    private func string(from codable: AnyCodable?) -> String? {
        guard let codable else { return nil }
        if let s = codable.value as? String, !s.isEmpty { return s }
        return nil
    }
}
