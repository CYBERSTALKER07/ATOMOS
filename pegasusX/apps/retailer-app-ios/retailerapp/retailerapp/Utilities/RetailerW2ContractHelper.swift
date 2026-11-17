import Foundation

extension RetailerMoney {
    /// Formats a minor-unit amount as rounded integer `so'm` (or specified currency code when non-UZS).
    /// Never leaks raw minor units (`tiyin`).
    static func formatWholeCurrency(_ minor: Int64, currency: String? = nil) -> String {
        let base = formatWholeSom(minor)
        let code = currency?.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() ?? ""
        if code.isEmpty || code == "UZS" || code == "SO'M" || code == "SOM" {
            return base
        }
        if base.hasSuffix(" so'm") {
            return String(base.dropLast(5)) + " " + code
        }
        return "\(base) \(code)"
    }

    static func somInputToMinor(_ text: String) -> Int64? {
        let stripped = text
            .replacingOccurrences(of: " ", with: "")
            .replacingOccurrences(of: "\u{00A0}", with: "")
            .replacingOccurrences(of: ",", with: "")
            .replacingOccurrences(of: "_", with: "")
            .replacingOccurrences(of: "'", with: "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard !stripped.isEmpty, stripped.allSatisfy({ $0.isNumber }), let som = Int64(stripped) else {
            return nil
        }
        guard som >= 0, som <= Int64.max / 100 else { return nil }
        return som * 100
    }

    static func minorToSomInput(_ minor: Int64) -> String {
        guard minor > 0 else { return "0" }
        return String((minor + 50) / 100)
    }
}

struct OwnerApprovalRequiredDetail: Equatable, Sendable {
    let requestId: String
    let clientOrderId: String?
    let status: String
    let trippedRule: String
    let amountMinor: Int64
    let thresholdMinor: Int64?
    let remainingCapMinor: Int64?
    let message: String
    let fallbackPaymentMethods: [String]

    var allowsCashFallback: Bool {
        fallbackPaymentMethods.contains { $0.uppercased() == "CASH" }
    }

    var allowsCardFallback: Bool {
        fallbackPaymentMethods.contains { $0.uppercased() == "CARD" }
    }

    var formattedAmount: String {
        RetailerMoney.formatWholeSom(amountMinor)
    }

    var formattedThreshold: String? {
        thresholdMinor.map { RetailerMoney.formatWholeSom($0) }
    }

    var formattedRemainingCap: String? {
        remainingCapMinor.map { RetailerMoney.formatWholeSom($0) }
    }

    var readableRuleLabel: String {
        switch trippedRule.uppercased() {
        case "DAILY_CREDIT_CAP_EXCEEDED":
            return "Daily credit cap exceeded"
        case "CREDIT_MANDATE_DISABLED":
            return "Credit purchases disabled by seller mandate"
        case "CASH_MANDATE_DISABLED":
            return "Cash purchases disabled by seller mandate"
        case "DAILY_CASH_CAP_EXCEEDED":
            return "Daily cash cap exceeded"
        case "MAX_SINGLE_ORDER_EXCEEDED":
            return "Single-order mandate limit exceeded"
        default:
            return trippedRule.replacingOccurrences(of: "_", with: " ").lowercased().capitalized
        }
    }
}

enum RetailerW2ErrorParser {
    static func parseOwnerApprovalRequired(from error: Error) -> OwnerApprovalRequiredDetail? {
        guard let apiError = error as? APIError else { return nil }

        let status: Int?
        var dict: [String: Any]?
        switch apiError {
        case .problemDetail(let problem):
            status = problem.status
            if let raw = problem.rawBody, let data = raw.data(using: .utf8),
               let parsed = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                dict = parsed
            } else if problem.code?.lowercased() == "owner_approval_required"
                        || problem.error?.lowercased() == "owner_approval_required" {
                return OwnerApprovalRequiredDetail(
                    requestId: "",
                    clientOrderId: nil,
                    status: "PENDING_OWNER",
                    trippedRule: "SELLER_MANDATE",
                    amountMinor: 0,
                    thresholdMinor: nil,
                    remainingCapMinor: nil,
                    message: problem.detail ?? problem.title ?? "Order exceeds seller mandate and requires owner approval.",
                    fallbackPaymentMethods: []
                )
            }
        case .serverError(let code, let body):
            status = code
            if let data = body.data(using: .utf8),
               let parsed = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                dict = parsed
            }
        default:
            return nil
        }

        guard status == 409, let dict else { return nil }
        let errCode = ((dict["code"] as? String) ?? (dict["error"] as? String) ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard errCode == "owner_approval_required" else { return nil }

        let requestId = (dict["request_id"] as? String) ?? ""
        let clientOrderId = dict["client_order_id"] as? String
        let approvalStatus = (dict["status"] as? String) ?? "PENDING_OWNER"
        let trippedRule = (dict["tripped_rule"] as? String) ?? "SELLER_MANDATE"
        let amountMinor = int64Value(dict["amount_minor"]) ?? 0
        let thresholdMinor = int64Value(dict["threshold_minor"])
        let remainingCapMinor = int64Value(dict["remaining_cap_minor"])
        let message = (dict["message"] as? String)
            ?? (dict["detail"] as? String)
            ?? "Order exceeds seller mandate and requires owner approval."
        let methods = (dict["fallback_payment_methods"] as? [Any])?
            .compactMap { $0 as? String }
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() }
            .filter { !$0.isEmpty } ?? []

        return OwnerApprovalRequiredDetail(
            requestId: requestId,
            clientOrderId: clientOrderId,
            status: approvalStatus,
            trippedRule: trippedRule,
            amountMinor: amountMinor,
            thresholdMinor: thresholdMinor,
            remainingCapMinor: remainingCapMinor,
            message: message,
            fallbackPaymentMethods: methods
        )
    }

    static func userFriendlyClaimError(for error: Error) -> String {
        let payload = extractJsonDict(from: error)
        let code = (
            (payload?["code"] as? String)
            ?? (payload?["error"] as? String)
            ?? ""
        )
        .trimmingCharacters(in: .whitespacesAndNewlines)
        .lowercased()

        let rawText: String = {
            if let apiError = error as? APIError {
                switch apiError {
                case .problemDetail(let p): return p.detail ?? p.title ?? ""
                case .serverError(_, let msg): return msg
                default: return apiError.localizedDescription
                }
            }
            return error.localizedDescription
        }()

        let effectiveCode: String = {
            if !code.isEmpty { return code }
            let lower = rawText.lowercased()
            if lower.contains("reused_photo_evidence_refused") { return "reused_photo_evidence_refused" }
            if lower.contains("photo_evidence_required") { return "photo_evidence_required" }
            if lower.contains("invalid_evidence_uri") { return "invalid_evidence_uri" }
            if lower.contains("claim_window_expired") { return "claim_window_expired" }
            if lower.contains("claim_not_allowed") { return "claim_not_allowed" }
            return ""
        }()

        switch effectiveCode {
        case "reused_photo_evidence_refused":
            return String(localized: "mobile_retailer.claims.error.reused_photo")
        case "photo_evidence_required":
            return String(localized: "mobile_retailer.claims.error.photo_required")
        case "invalid_evidence_uri":
            return String(localized: "mobile_retailer.claims.error.invalid_photo")
        case "claim_window_expired":
            return String(localized: "mobile_retailer.claims.error.window_expired")
        case "claim_not_allowed":
            return (payload?["message"] as? String)
                ?? (payload?["detail"] as? String)
                ?? "Claim cannot be filed for this order."
        default:
            if let message = payload?["message"] as? String, !message.isEmpty {
                return message
            }
            if let detail = payload?["detail"] as? String, !detail.isEmpty {
                return detail
            }
            return error.localizedDescription
        }
    }

    private static func extractJsonDict(from error: Error) -> [String: Any]? {
        guard let apiError = error as? APIError else { return nil }
        switch apiError {
        case .problemDetail(let p):
            if let raw = p.rawBody, let data = raw.data(using: .utf8),
               let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                return dict
            }
            var dict: [String: Any] = [:]
            if let code = p.code { dict["code"] = code }
            if let err = p.error { dict["error"] = err }
            if let detail = p.detail { dict["detail"] = detail }
            return dict.isEmpty ? nil : dict
        case .serverError(_, let msg):
            if let data = msg.data(using: .utf8),
               let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                return dict
            }
            return nil
        default:
            return nil
        }
    }

    private static func int64Value(_ raw: Any?) -> Int64? {
        switch raw {
        case let v as Int64: return v
        case let v as Int: return Int64(v)
        case let v as Double: return Int64(v.rounded())
        case let v as NSNumber: return v.int64Value
        default: return nil
        }
    }
}
