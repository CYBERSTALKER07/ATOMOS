import Foundation

enum PackSnapMode: Sendable, Hashable {
    case nearest
    case floor
    case ceil
}

struct PackMultipleSuggestion: Identifiable, Hashable, Sendable {
    let id: String
    let skuId: String
    let skuName: String?
    let packSize: Int
    let requested: Int
    let suggestedFloor: Int
    let suggestedCeil: Int
    let suggestedNearest: Int

    init(
        skuId: String,
        skuName: String? = nil,
        packSize: Int,
        requested: Int,
        suggestedFloor: Int,
        suggestedCeil: Int,
        suggestedNearest: Int? = nil
    ) {
        self.id = skuId
        self.skuId = skuId
        self.skuName = skuName
        let safePack = max(1, packSize)
        self.packSize = safePack
        self.requested = requested
        self.suggestedFloor = suggestedFloor
        self.suggestedCeil = suggestedCeil
        self.suggestedNearest = suggestedNearest
            ?? RetailerPackHelper.snapToPackMultiple(requested, packSize: safePack, mode: .nearest)
    }
}

extension RetailerMoney {
    private static let somGroupingFormatter: NumberFormatter = {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.groupingSeparator = " "
        formatter.maximumFractionDigits = 0
        formatter.usesGroupingSeparator = true
        return formatter
    }()

    /// Rounds minor units (tiyin) to integer so'm (`minor / 100`, half-up away from zero), preventing `-0`.
    static func roundMinorToSom(_ minor: Int64) -> Int64 {
        let magnitude: Int64 = minor == Int64.min ? Int64.max : abs(minor)
        let roundedAbs = (magnitude + 50) / 100
        if roundedAbs == 0 {
            return 0
        }
        return minor < 0 ? -roundedAbs : roundedAbs
    }

    static func formatWholeSom(_ minor: Int64) -> String {
        let som = roundMinorToSom(minor)
        let formatted = somGroupingFormatter.string(from: NSNumber(value: som)) ?? "\(som)"
        return "\(formatted) so'm"
    }

    /// Formats a signed substitute `priceGapMinor` delta in rounded integer `so'm` (`+250 so'm`, `-120 so'm`, `0 so'm`).
    static func formatPriceGapSom(_ priceGapMinor: Int64) -> String {
        let som = roundMinorToSom(priceGapMinor)
        if som > 0 {
            let formatted = somGroupingFormatter.string(from: NSNumber(value: som)) ?? "\(som)"
            return "+\(formatted) so'm"
        } else if som < 0 {
            let formatted = somGroupingFormatter.string(from: NSNumber(value: -som)) ?? "\(-som)"
            return "-\(formatted) so'm"
        } else {
            return "0 so'm"
        }
    }
}

enum RetailerPackHelper {
    static func effectivePackSize(product: Product? = nil, variant: Variant? = nil) -> Int {
        if let variantPack = variant?.packCount, variantPack > 1 {
            return variantPack
        }
        if product?.sellByBlock == true, let units = product?.unitsPerBlock, units > 1 {
            return units
        }
        return max(1, variant?.packCount ?? 1)
    }

    static func snapToPackMultiple(_ quantity: Int, packSize: Int, mode: PackSnapMode = .nearest) -> Int {
        guard quantity > 0 else { return 0 }
        let safePack = max(1, packSize)
        if safePack == 1 { return quantity }
        let remainder = quantity % safePack
        if remainder == 0 { return quantity }

        let lower = quantity - remainder
        let safeFloor = lower >= safePack ? lower : safePack
        let safeCeil = ((quantity + safePack - 1) / safePack) * safePack

        switch mode {
        case .floor:
            return safeFloor
        case .ceil:
            return safeCeil
        case .nearest:
            if lower < safePack {
                return safeCeil
            }
            let downDist = quantity - safeFloor
            let upDist = safeCeil - quantity
            return upDist <= downDist ? safeCeil : safeFloor
        }
    }
}

enum PendingOrderQueueDisposition: Equatable, Sendable {
    case parkAwaitingOwnerApproval
    case parkRequestInProgress
    case parkRateLimited
    case parkAuthRequired
    case purgeIdempotentDuplicate
    case purgeRejectedValidation
    case retryTransient
}

struct PendingOrderErrorAssessment: Equatable, Sendable {
    let disposition: PendingOrderQueueDisposition
    let queueStatus: String
    let userMessage: String
    let shouldNotify: Bool
}

enum RetailerOrderContractHelper {
    private static let idempotentCreatedCodes: Set<String> = [
        "already_created",
        "idempotent_replay",
        "duplicate_request",
        "order_already_exists",
        "idempotency_conflict_completed",
    ]

    private static let oosAndValidation409Codes: Set<String> = [
        "inventory_exhausted",
        "ALL_ITEMS_OUT_OF_STOCK",
        "PARTIAL_OUT_OF_STOCK_REJECTED",
        "order_acceptance_closed",
        "PACK_MULTIPLE_REQUIRED",
    ]

    static func extractStatusCodeAndPayload(from error: Error) -> (status: Int?, code: String?, rawJson: [String: Any]?) {
        if let apiError = error as? APIError {
            switch apiError {
            case .problemDetail(let problem):
                let parsedRaw: [String: Any]? = problem.rawBody
                    .flatMap { $0.data(using: .utf8) }
                    .flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] }
                let code = problem.code
                    ?? problem.error
                    ?? parsedRaw?["code"] as? String
                    ?? parsedRaw?["error"] as? String
                return (problem.status, code, parsedRaw)
            case .serverError(let statusCode, let message):
                let parsedRaw: [String: Any]? = message.data(using: .utf8)
                    .flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] }
                let code = parsedRaw?["code"] as? String ?? parsedRaw?["error"] as? String
                return (statusCode, code, parsedRaw)
            default:
                return (nil, nil, nil)
            }
        }
        return (nil, nil, nil)
    }

    static func parsePackMultipleSuggestions(
        from error: Error,
        skuNamesBySku: [String: String] = [:]
    ) -> [PackMultipleSuggestion] {
        let extracted = extractStatusCodeAndPayload(from: error)
        let normalizedCode = extracted.code?.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()

        if let apiError = error as? APIError, case .problemDetail(let problem) = apiError {
            let problemCode = (problem.code ?? problem.error ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
            let hasProblemPackFields = (problem.packSize ?? 0) > 1 || problem.suggestedFloor != nil || problem.suggestedCeil != nil || problem.suggestedNearest != nil
            if problemCode == "PACK_MULTIPLE_REQUIRED", let sku = problem.skuId, !sku.isEmpty, hasProblemPackFields {
                let packSize = max(1, problem.packSize ?? 1)
                let requested = max(1, problem.requested ?? 1)
                let floorVal = problem.suggestedFloor ?? RetailerPackHelper.snapToPackMultiple(requested, packSize: packSize, mode: .floor)
                let ceilVal = problem.suggestedCeil ?? RetailerPackHelper.snapToPackMultiple(requested, packSize: packSize, mode: .ceil)
                let nearestVal = problem.suggestedNearest ?? RetailerPackHelper.snapToPackMultiple(requested, packSize: packSize, mode: .nearest)
                return [
                    PackMultipleSuggestion(
                        skuId: sku,
                        skuName: skuNamesBySku[sku],
                        packSize: packSize,
                        requested: requested,
                        suggestedFloor: floorVal,
                        suggestedCeil: ceilVal,
                        suggestedNearest: nearestVal
                    ),
                ]
            }
        }

        guard let json = extracted.rawJson else { return [] }
        let rawCode = (json["code"] as? String ?? json["error"] as? String ?? normalizedCode ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .uppercased()
        guard rawCode == "PACK_MULTIPLE_REQUIRED" else { return [] }

        var results: [PackMultipleSuggestion] = []
        if let list = (json["suggestions"] as? [[String: Any]])
            ?? (json["line_errors"] as? [[String: Any]])
            ?? (json["items"] as? [[String: Any]])
            ?? (json["details"] as? [[String: Any]])
        {
            for entry in list {
                if let suggestion = suggestion(from: entry, skuNamesBySku: skuNamesBySku) {
                    results.append(suggestion)
                }
            }
        }
        if results.isEmpty, let single = suggestion(from: json, skuNamesBySku: skuNamesBySku) {
            results.append(single)
        }
        return results
    }

    private static func suggestion(
        from dict: [String: Any],
        skuNamesBySku: [String: String]
    ) -> PackMultipleSuggestion? {
        let sku = (dict["sku_id"] as? String)
            ?? (dict["product_id"] as? String)
            ?? (dict["sku"] as? String)
            ?? ""
        guard !sku.isEmpty else { return nil }
        let packSize = max(1, intValue(dict["pack_size"]) ?? 1)
        let requested = max(1, intValue(dict["requested"]) ?? intValue(dict["requested_qty"]) ?? 1)
        let floorVal = intValue(dict["suggested_floor"])
            ?? RetailerPackHelper.snapToPackMultiple(requested, packSize: packSize, mode: .floor)
        let ceilVal = intValue(dict["suggested_ceil"])
            ?? RetailerPackHelper.snapToPackMultiple(requested, packSize: packSize, mode: .ceil)
        let nearestVal = intValue(dict["suggested_nearest"])
            ?? RetailerPackHelper.snapToPackMultiple(requested, packSize: packSize, mode: .nearest)
        return PackMultipleSuggestion(
            skuId: sku,
            skuName: skuNamesBySku[sku] ?? (dict["sku_name"] as? String) ?? (dict["product_name"] as? String),
            packSize: packSize,
            requested: requested,
            suggestedFloor: floorVal,
            suggestedCeil: ceilVal,
            suggestedNearest: nearestVal
        )
    }

    private static func intValue(_ any: Any?) -> Int? {
        if let intVal = any as? Int { return intVal }
        if let int64Val = any as? Int64 { return Int(int64Val) }
        if let numVal = any as? NSNumber { return numVal.intValue }
        if let strVal = any as? String { return Int(strVal) }
        return nil
    }

    static func assessPendingOrderError(_ error: Error) -> PendingOrderErrorAssessment {
        if RetailerErrorSupport.classify(error) == .offline {
            return PendingOrderErrorAssessment(
                disposition: .retryTransient,
                queueStatus: PendingOrder.statusPending,
                userMessage: "Saved to sync queue. Will replay automatically when back online.",
                shouldNotify: false
            )
        }

        let (status, rawCode, rawJson) = extractStatusCodeAndPayload(from: error)
        let normalizedCode = (rawCode ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()

        if status == 429 || normalizedCode == "rate_limited" || normalizedCode == "too_many_requests" {
            return PendingOrderErrorAssessment(
                disposition: .parkRateLimited,
                queueStatus: PendingOrder.statusRateLimited,
                userMessage: "Rate limited (429) — order stays in sync queue and will retry shortly.",
                shouldNotify: true
            )
        }

        if status == 409 {
            let detailText = (
                (rawJson?["detail"] as? String)
                    ?? (rawJson?["message"] as? String)
                    ?? ""
            ).lowercased()

            if normalizedCode == "owner_approval_required"
                || (normalizedCode.isEmpty && detailText.contains("owner") && detailText.contains("approval"))
            {
                return PendingOrderErrorAssessment(
                    disposition: .parkAwaitingOwnerApproval,
                    queueStatus: PendingOrder.statusAwaitingOwnerApproval,
                    userMessage: "Awaiting owner approval (409) — kept safely in sync queue.",
                    shouldNotify: true
                )
            }

            if normalizedCode == "request_in_progress"
                || (normalizedCode.isEmpty && detailText.contains("in progress"))
            {
                return PendingOrderErrorAssessment(
                    disposition: .parkRequestInProgress,
                    queueStatus: PendingOrder.statusInProgress,
                    userMessage: "Order request already in progress (409) — kept in sync queue until finalized.",
                    shouldNotify: true
                )
            }

            if idempotentCreatedCodes.contains(normalizedCode) {
                return PendingOrderErrorAssessment(
                    disposition: .purgeIdempotentDuplicate,
                    queueStatus: PendingOrder.statusCompleted,
                    userMessage: "Order already created on server.",
                    shouldNotify: false
                )
            }

            if let rawCode, oosAndValidation409Codes.contains(rawCode) || oosAndValidation409Codes.contains(rawCode.uppercased()) {
                return PendingOrderErrorAssessment(
                    disposition: .purgeRejectedValidation,
                    queueStatus: PendingOrder.statusFailed,
                    userMessage: (rawJson?["detail"] as? String)
                        ?? (rawJson?["message"] as? String)
                        ?? "Order rejected due to stock or pack constraints.",
                    shouldNotify: true
                )
            }

            // Default remaining 409 conflict (e.g. uncoded idempotent conflict already completed)
            return PendingOrderErrorAssessment(
                disposition: .purgeIdempotentDuplicate,
                queueStatus: PendingOrder.statusCompleted,
                userMessage: "Conflict resolved on server; removed from queue.",
                shouldNotify: false
            )
        }

        if status == 401 || normalizedCode == "unauthorized" || normalizedCode == "unauthenticated" {
            return PendingOrderErrorAssessment(
                disposition: .parkAuthRequired,
                queueStatus: PendingOrder.statusAuthRequired,
                userMessage: "Session expired (401) — sign in again to sync queued order.",
                shouldNotify: true
            )
        }

        if let status, (400...499).contains(status), status != 408 && status != 401 && status != 403 {
            return PendingOrderErrorAssessment(
                disposition: .purgeRejectedValidation,
                queueStatus: PendingOrder.statusFailed,
                userMessage: "Queued order rejected (HTTP \(status)).",
                shouldNotify: true
            )
        }

        return PendingOrderErrorAssessment(
            disposition: .retryTransient,
            queueStatus: PendingOrder.statusPending,
            userMessage: RetailerErrorSupport.retryQueuedMessage(
                for: error,
                fallback: "Saved to sync queue. Retry scheduled."
            ),
            shouldNotify: false
        )
    }
}
