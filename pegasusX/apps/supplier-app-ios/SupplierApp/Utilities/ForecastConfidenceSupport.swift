import Foundation
import SwiftUI

/// Canonical mapper — keep aligned with packages/types/forecast-confidence.ts
enum ForecastConfidenceSupport {
    private static let staleMinutes: Int64 = 30

    static func isForecastStale(generatedAt: String?) -> Bool {
        guard let generatedAt, let date = ISO8601DateFormatter().date(from: generatedAt) else { return false }
        return Int64(Date().timeIntervalSince(date) / 60) > staleMinutes
    }

    static func formatForecastUpdatedAt(generatedAt: String?) -> String? {
        guard let generatedAt, let date = ISO8601DateFormatter().date(from: generatedAt) else {
            return generatedAt
        }
        let mins = Int64(Date().timeIntervalSince(date) / 60)
        if mins < 1 { return "just now" }
        if mins < 60 { return "\(mins)m ago" }
        return generatedAt
    }

    static func mapBaselineSource(_ src: String?) -> String? {
        switch src {
        case "demand_forecast_baseline": return "moving_average"
        case "ai_recommendations", "inventory_hint": return "inventory_hint"
        case "ml": return "moving_average"
        case "mixed": return "mixed"
        default: return src
        }
    }

    static func formatBaselineSourceLabel(_ src: String?) -> String {
        switch mapBaselineSource(src) {
        case "moving_average": return "Baseline"
        case "seasonal_template": return "Seasonal"
        case "inventory_hint": return "Hint"
        case "mixed": return "Mixed"
        default:
            return mapBaselineSource(src)?.replacingOccurrences(of: "_", with: " ") ?? ""
        }
    }

    static func fromDemand(_ summary: SupplierDemandSummaryResponse) -> ForecastConfidence {
        if let confidence = summary.confidence {
            return confidence
        }
        if summary.predictionCount == 0 {
            return ForecastConfidence(
                lowUnits: nil,
                highUnits: nil,
                confidencePct: nil,
                baselineSource: mapBaselineSource(summary.baselineSource),
                blockedReason: "no_predictions",
                label: "insufficient_history"
            )
        }
        let mid = summary.totalPallets
        let spread = max(1, Int((Double(mid) * 0.1).rounded()))
        let src = mapBaselineSource(summary.baselineSource)
        var confidence = 65
        if src == "seasonal_template" {
            confidence = 75
        } else if src == "inventory_hint" {
            confidence = 72
        } else if summary.predictionCount >= 5 {
            confidence = 72
        }
        return ForecastConfidence(
            lowUnits: Int64(max(0, mid - spread)),
            highUnits: Int64(mid + spread),
            confidencePct: confidence,
            baselineSource: src,
            blockedReason: nil,
            label: summary.predictionCount < 3 ? "early_signal" : "standard"
        )
    }
}

enum PlanBrainTab: String, CaseIterable {
    case planning
    case brain
}

func planBrainTabFromQuery(_ raw: String?) -> PlanBrainTab {
    raw?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "brain" ? .brain : .planning
}

func brainForecastLine(confidence: ForecastConfidence?, accuracyPoints: [Double]) -> [Double]? {
    if confidence?.isBlocked == true { return nil }
    if accuracyPoints.count < 2 { return nil }
    return accuracyPoints
}

func factoryPlanningDisabledCode(status: Int, body: String) -> String? {
    guard status == 409, body.contains("factory_planning_disabled") else { return nil }
    return "factory_planning_disabled"
}

extension ForecastConfidence {
    var isBlocked: Bool {
        blockedReason != nil || label == "insufficient_history"
    }

    var confidenceTint: Color {
        guard let pct = confidencePct else { return SupplierTheme.secondaryLabel }
        if pct >= 80 { return SupplierTheme.success }
        if pct >= 60 { return SupplierTheme.warning }
        return SupplierTheme.destructive
    }
}
