import SwiftUI

struct ForecastConfidenceData {
    let lowUnits: Int64?
    let highUnits: Int64?
    let confidencePct: Int?
    let baselineSource: String?
    let blockedReason: String?
    let label: String?

    var blocked: Bool {
        if let blockedReason, !blockedReason.isEmpty { return true }
        return label == "insufficient_history"
    }

    var seasonalActive: Bool {
        baselineSource == "seasonal_template"
    }
}

func parseForecastConfidence(_ breakdown: [String: AnyCodable]?) -> ForecastConfidenceData? {
    guard let breakdown, !breakdown.isEmpty else { return nil }
    let low = int64(from: breakdown["low_units"])
    let high = int64(from: breakdown["high_units"])
    var confidencePct = int(from: breakdown["confidence_pct"])
    if confidencePct == nil, let raw = number(from: breakdown["confidence"]) {
        confidencePct = raw <= 1 ? Int(raw * 100) : Int(raw)
    }
    let baselineSource = string(from: breakdown["baseline_source"])
    let blockedReason = string(from: breakdown["blocked_reason"])
    let label = string(from: breakdown["label"])
    let predicted = int64(from: breakdown["predictedQty"]) ?? int64(from: breakdown["predicted_qty"])
    let derivedLow = low ?? predicted.map { Int64((Double($0) * 0.9).rounded(.down)) }
    let derivedHigh = high ?? predicted.map { Int64((Double($0) * 1.1).rounded(.up)) }

    if derivedLow == nil, derivedHigh == nil, confidencePct == nil,
       blockedReason == nil, baselineSource == nil, label == nil {
        return nil
    }
    return ForecastConfidenceData(
        lowUnits: derivedLow,
        highUnits: derivedHigh ?? derivedLow,
        confidencePct: confidencePct,
        baselineSource: baselineSource,
        blockedReason: blockedReason,
        label: label
    )
}

func formatSourceBadge(_ source: String?) -> String {
    guard let source, !source.isEmpty else { return "" }
    switch source {
    case "ml": return "ML"
    case "moving_average": return "Baseline"
    case "seasonal_template": return "Seasonal"
    case "mixed": return "Mixed"
    default: return source.replacingOccurrences(of: "_", with: " ")
    }
}

private func number(from codable: AnyCodable?) -> Double? {
    guard let codable else { return nil }
    if let d = codable.value as? Double { return d }
    if let i = codable.value as? Int { return Double(i) }
    if let i = codable.value as? Int64 { return Double(i) }
    if let s = codable.value as? String, let d = Double(s) { return d }
    return nil
}

private func int64(from codable: AnyCodable?) -> Int64? {
    guard let codable else { return nil }
    if let i = codable.value as? Int64 { return i }
    if let i = codable.value as? Int { return Int64(i) }
    if let d = codable.value as? Double { return Int64(d) }
    if let s = codable.value as? String, let i = Int64(s) { return i }
    return nil
}

private func int(from codable: AnyCodable?) -> Int? {
    guard let codable else { return nil }
    if let i = codable.value as? Int { return i }
    if let i = codable.value as? Int64 { return Int(i) }
    if let d = codable.value as? Double { return Int(d) }
    if let s = codable.value as? String, let i = Int(s) { return i }
    return nil
}

private func string(from codable: AnyCodable?) -> String? {
    guard let codable else { return nil }
    if let s = codable.value as? String, !s.isEmpty { return s }
    return nil
}

struct ForecastConfidenceView: View {
    let confidence: ForecastConfidenceData
    var compact: Bool = false

    var body: some View {
        if compact {
            HStack(spacing: 6) {
                if confidence.blocked {
                    Text("Insufficient history")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(LabTheme.warning)
                } else {
                    let low = confidence.lowUnits ?? 0
                    let high = confidence.highUnits ?? low
                    Text("\(formatUnits(low)) – \(formatUnits(high))")
                        .font(.caption.monospacedDigit())
                }
                if let source = confidence.baselineSource, !source.isEmpty {
                    Text(formatSourceBadge(source))
                        .font(.caption2.weight(.semibold))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(.quaternary)
                        .clipShape(Capsule())
                }
                if confidence.seasonalActive {
                    Text("Seasonal")
                        .font(.caption2)
                        .foregroundStyle(LabTheme.warning)
                }
            }
        } else {
            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    Text("Forecast confidence")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                    Spacer()
                    if let source = confidence.baselineSource, !source.isEmpty {
                        Text(formatSourceBadge(source))
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(.quaternary)
                            .clipShape(Capsule())
                    }
                }
                if confidence.blocked {
                    Text("Insufficient history — predictive forecast blocked")
                        .font(.subheadline)
                        .foregroundStyle(LabTheme.warning)
                } else {
                    let low = confidence.lowUnits ?? 0
                    let high = confidence.highUnits ?? low
                    Text("\(formatUnits(low)) – \(formatUnits(high)) units")
                        .font(.subheadline.bold().monospacedDigit())
                }
                if let pct = confidence.confidencePct, !confidence.blocked {
                    Text("\(pct)% confidence")
                        .font(.caption)
                        .foregroundStyle(confidenceColor(pct))
                }
                if confidence.seasonalActive {
                    Text("Seasonal template active")
                        .font(.caption)
                        .foregroundStyle(LabTheme.warning)
                }
            }
            .padding(12)
            .labCard()
        }
    }

    private func confidenceColor(_ pct: Int) -> Color {
        if pct >= 80 { return LabTheme.success }
        if pct >= 60 { return LabTheme.warning }
        return LabTheme.destructive
    }

    private func formatUnits(_ value: Int64) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter.string(from: NSNumber(value: value)) ?? "\(value)"
    }
}
