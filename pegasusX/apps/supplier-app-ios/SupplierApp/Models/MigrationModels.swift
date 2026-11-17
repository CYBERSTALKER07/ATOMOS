import Foundation

enum MigrationPack: String, Codable, CaseIterable, Identifiable {
    case warehouses
    case productsPrices = "products_prices"
    case customers
    case stock
    case lots
    case historicalOrders = "historical_orders"
    case nasiyaAR = "nasiya_ar"
    case openOrders = "open_orders"

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .warehouses: "mobile_supplier.ui.warehouse"
        case .productsPrices: "supplier_portal.migrations.pack_products"
        case .customers: "supplier_portal.migrations.pack_retailers"
        case .stock: "supplier_portal.migrations.pack_stock"
        case .lots: "supplier_portal.migrations.pack_lots"
        case .historicalOrders: "supplier_portal.migrations.pack_orders_history"
        case .nasiyaAR: "supplier_portal.migrations.pack_nasiya"
        case .openOrders: "supplier_portal.migrations.pack_orders_open"
        }
    }

    static func labelKey(for raw: String) -> String {
        MigrationPack(rawValue: raw)?.labelKey ?? "supplier_portal.migrations.pack_quarantine"
    }
}

enum MigrationEncodingOption {
    static let all = ["utf-8", "windows-1251", "cp1251", "cp866"]
}

struct MigrationProject: Decodable, Identifiable, Hashable {
    var id: String { projectId }
    let supplierId: String
    let projectId: String
    let name: String
    let mode: String
    let status: String
    let defaultWarehouseId: String?
    let createdBy: String
    let cutoverAt: String?
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case projectId = "project_id"
        case name, mode, status
        case defaultWarehouseId = "default_warehouse_id"
        case createdBy = "created_by"
        case cutoverAt = "cutover_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct MigrationProjectsResponse: Decodable {
    let projects: [MigrationProject]
}

struct MigrationSortedStreamDecision: Decodable, Identifiable, Equatable {
    var id: String { "\(artifactId)::\(streamKey)" }
    let artifactId: String
    let fileName: String
    let streamKey: String
    let encoding: String
    let totalRows: Int64
    let headers: [String]
    let previewRaw: [String]?
    let detectedPacks: [String]
    let splitKeys: [String]
    let confidence: Double
    let isComposite: Bool
    let ignored: Bool?
    let needsReview: Bool
    let reviewReason: String?

    enum CodingKeys: String, CodingKey {
        case artifactId = "artifact_id"
        case fileName = "file_name"
        case streamKey = "stream_key"
        case encoding
        case totalRows = "total_rows"
        case headers
        case previewRaw = "preview_raw"
        case detectedPacks = "detected_packs"
        case splitKeys = "split_keys"
        case confidence
        case isComposite = "is_composite"
        case ignored
        case needsReview = "needs_review"
        case reviewReason = "review_reason"
    }
}

struct MigrationSortSummary: Decodable, Equatable {
    let streams: [MigrationSortedStreamDecision]
    let conflictingInventoryMode: Bool
    let needsReview: Bool
    let reviewReasons: [String]?

    enum CodingKeys: String, CodingKey {
        case streams
        case conflictingInventoryMode = "conflicting_inventory_mode"
        case needsReview = "needs_review"
        case reviewReasons = "review_reasons"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        streams = try c.decodeIfPresent([MigrationSortedStreamDecision].self, forKey: .streams) ?? []
        conflictingInventoryMode = try c.decodeIfPresent(Bool.self, forKey: .conflictingInventoryMode) ?? false
        needsReview = try c.decodeIfPresent(Bool.self, forKey: .needsReview) ?? false
        reviewReasons = try c.decodeIfPresent([String].self, forKey: .reviewReasons)
    }
}

struct MigrationBundle: Decodable, Identifiable {
    var id: String { bundleId }
    let supplierId: String
    let projectId: String
    let bundleId: String
    let status: String
    let sortSummary: MigrationSortSummary?
    let errorMessage: String?
    let createdBy: String
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case projectId = "project_id"
        case bundleId = "bundle_id"
        case status
        case sortSummary = "sort_summary"
        case errorMessage = "error_message"
        case createdBy = "created_by"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct MigrationArtifact: Decodable, Identifiable {
    var id: String { artifactId }
    let artifactId: String
    let fileName: String
    let byteSize: Int64
    let declaredMime: String?
    let detectedMime: String?
    let contentSha256: String
    let encoding: String?
    let gcsPath: String

    enum CodingKeys: String, CodingKey {
        case artifactId = "artifact_id"
        case fileName = "file_name"
        case byteSize = "byte_size"
        case declaredMime = "declared_mime"
        case detectedMime = "detected_mime"
        case contentSha256 = "content_sha256"
        case encoding
        case gcsPath = "gcs_path"
    }
}

struct MigrationBundleTree: Decodable, Identifiable {
    var id: String { bundle.bundleId }
    let bundle: MigrationBundle
    let artifacts: [MigrationArtifact]
}

struct MigrationSession: Decodable, Identifiable, Hashable {
    var id: String { sessionId }
    let supplierId: String
    let sessionId: String
    let status: String
    let fileName: String
    let totalRows: Int64
    let projectId: String?
    let bundleId: String?
    let artifactId: String?
    let streamKey: String?
    let streamSplitKey: String?
    let pack: String?
    let sheetName: String?
    let aiModel: String?
    let aiStatus: String?
    let dryRunReportId: String?
    let dryRunFingerprint: String?
    let applyMode: String?
    let setOnHandConfirmed: Bool?
    let resumeRowIndex: Int64?
    let rollbackUntil: String?
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case sessionId = "session_id"
        case status
        case fileName = "file_name"
        case totalRows = "total_rows"
        case projectId = "project_id"
        case bundleId = "bundle_id"
        case artifactId = "artifact_id"
        case streamKey = "stream_key"
        case streamSplitKey = "stream_split_key"
        case pack
        case sheetName = "sheet_name"
        case aiModel = "ai_model"
        case aiStatus = "ai_status"
        case dryRunReportId = "dry_run_report_id"
        case dryRunFingerprint = "dry_run_fingerprint"
        case applyMode = "apply_mode"
        case setOnHandConfirmed = "set_on_hand_confirmed"
        case resumeRowIndex = "resume_row_index"
        case rollbackUntil = "rollback_until"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct MigrationDAGStatus: Decodable {
    let order: [String]
    let runnableNow: [String]
    let blockedBy: [String: [String]]
    let missingPacks: [String]?

    enum CodingKeys: String, CodingKey {
        case order
        case runnableNow = "runnable_now"
        case blockedBy = "blocked_by"
        case missingPacks = "missing_packs"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        order = try c.decodeIfPresent([String].self, forKey: .order) ?? []
        runnableNow = try c.decodeIfPresent([String].self, forKey: .runnableNow) ?? []
        blockedBy = try c.decodeIfPresent([String: [String]].self, forKey: .blockedBy) ?? [:]
        missingPacks = try c.decodeIfPresent([String].self, forKey: .missingPacks)
    }
}

struct MigrationProjectTree: Decodable {
    let project: MigrationProject
    let bundles: [MigrationBundleTree]
    let sessions: [MigrationSession]
    let dag: MigrationDAGStatus
}

struct MigrationDeclareFile: Encodable {
    let fileName: String
    let byteSize: Int64
    let contentSha256: String

    enum CodingKeys: String, CodingKey {
        case fileName = "file_name"
        case byteSize = "byte_size"
        case contentSha256 = "content_sha256"
    }
}

struct MigrationCreateBundleRequest: Encodable {
    let files: [MigrationDeclareFile]
}

struct MigrationUploadTicket: Decodable, Identifiable {
    var id: String { artifactId }
    let artifactId: String
    let fileName: String
    let gcsPath: String
    let uploadUrl: String
    let method: String
    let contentType: String
    let resumable: Bool
    let requiredHeaders: [String: String]?
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case artifactId = "artifact_id"
        case fileName = "file_name"
        case gcsPath = "gcs_path"
        case uploadUrl = "upload_url"
        case method
        case contentType = "content_type"
        case resumable
        case requiredHeaders = "required_headers"
        case expiresAt = "expires_at"
    }
}

struct MigrationCreateBundleResponse: Decodable {
    let bundle: MigrationBundle
    let artifacts: [MigrationArtifact]
    let uploadTickets: [MigrationUploadTicket]

    enum CodingKeys: String, CodingKey {
        case bundle, artifacts
        case uploadTickets = "upload_tickets"
    }
}

struct MigrationUploadedArtifactMeta: Encodable {
    let byteSize: Int64
    let contentSha256: String

    enum CodingKeys: String, CodingKey {
        case byteSize = "byte_size"
        case contentSha256 = "content_sha256"
    }
}

struct MigrationBundleUploadedRequest: Encodable {
    let artifacts: [String: MigrationUploadedArtifactMeta]
}

struct MigrationBundleUploadedResponse: Decodable {
    let bundle: MigrationBundle
    let summary: MigrationSortSummary?
    let sessions: [MigrationSession]?
}

struct MigrationStreamOverrideItem: Encodable {
    let artifactId: String
    let streamKey: String
    let packs: [String]?
    let ignore: Bool?
    let encoding: String?
    let confirmed: Bool?

    enum CodingKeys: String, CodingKey {
        case artifactId = "artifact_id"
        case streamKey = "stream_key"
        case packs, ignore, encoding, confirmed
    }
}

struct MigrationBundleSortOverrideRequest: Encodable {
    let overrides: [MigrationStreamOverrideItem]
    let resolvedInventoryMode: String?
    let confirmReview: Bool

    enum CodingKeys: String, CodingKey {
        case overrides
        case resolvedInventoryMode = "resolved_inventory_mode"
        case confirmReview = "confirm_review"
    }
}

struct MigrationBundleSortOverrideResponse: Decodable {
    let bundleId: String
    let bundleStatus: String
    let summary: MigrationSortSummary
    let sessions: [MigrationSession]

    enum CodingKeys: String, CodingKey {
        case bundleId = "bundle_id"
        case bundleStatus = "bundle_status"
        case summary, sessions
    }
}

struct MigrationCanonicalField: Decodable, Identifiable, Hashable {
    var id: String { key }
    let key: String
    let labelEn: String
    let labelRu: String?
    let labelUz: String?
    let required: Bool
    let type: String

    enum CodingKeys: String, CodingKey {
        case key
        case labelEn = "label_en"
        case labelRu = "label_ru"
        case labelUz = "label_uz"
        case required, type
    }

    var displayLabel: String {
        let code = Locale.current.language.languageCode?.identifier ?? "en"
        switch code {
        case "ru": return (labelRu?.isEmpty == false ? labelRu : labelEn) ?? key
        case "uz": return (labelUz?.isEmpty == false ? labelUz : labelEn) ?? key
        default: return labelEn.isEmpty ? key : labelEn
        }
    }
}

struct MigrationColumnMappingProposal: Codable, Identifiable, Equatable {
    var id: String { sourceColumn }
    let sourceColumn: String
    var targetField: String
    var confidence: Double
    var reason: String?
    var source: String?
    var confirmed: Bool
    var transform: String?

    enum CodingKeys: String, CodingKey {
        case sourceColumn = "source_column"
        case targetField = "target_field"
        case confidence, reason, source, confirmed, transform
    }
}

struct MigrationSessionMappingDocument: Decodable {
    let sessionId: String
    let pack: String
    let headerFingerprint: String
    let aiModel: String
    let aiStatus: String
    let mappings: [MigrationColumnMappingProposal]
    let missingRequired: [String]?
    let lowConfidenceColumns: [String]?
    let allRequiredMapped: Bool
    let mappingHash: String

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case pack
        case headerFingerprint = "header_fingerprint"
        case aiModel = "ai_model"
        case aiStatus = "ai_status"
        case mappings
        case missingRequired = "missing_required"
        case lowConfidenceColumns = "low_confidence_columns"
        case allRequiredMapped = "all_required_mapped"
        case mappingHash = "mapping_hash"
    }
}

struct MigrationGetSessionMappingResponse: Decodable {
    let session: MigrationSession
    let taxonomy: [MigrationCanonicalField]
    let mapping: MigrationSessionMappingDocument
}

struct MigrationConfirmMappingRequest: Encodable {
    let mappings: [MigrationColumnMappingProposal]?
    let acceptAboveConfidence: Double?
    let saveToLibrary: Bool?
    let strict: Bool

    enum CodingKeys: String, CodingKey {
        case mappings
        case acceptAboveConfidence = "accept_above_confidence"
        case saveToLibrary = "save_to_library"
        case strict
    }
}

struct MigrationConfirmMappingResponse: Decodable {
    let session: MigrationSession
    let mapping: MigrationSessionMappingDocument
}

enum FlexibleJSONValue: Decodable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case null

    init(from decoder: Decoder) throws {
        let c = try decoder.singleValueContainer()
        if c.decodeNil() {
            self = .null
        } else if let s = try? c.decode(String.self) {
            self = .string(s)
        } else if let b = try? c.decode(Bool.self) {
            self = .bool(b)
        } else if let d = try? c.decode(Double.self) {
            self = .number(d)
        } else {
            self = .null
        }
    }

    var text: String {
        switch self {
        case .string(let s): return s
        case .number(let d):
            if d.rounded() == d && abs(d) < Double(Int64.max) {
                return String(Int64(d))
            }
            return String(d)
        case .bool(let b): return b ? "true" : "false"
        case .null: return ""
        }
    }
}

struct MigrationStagedRow: Decodable, Identifiable {
    var id: Int64 { rowIndex }
    let rowIndex: Int64
    let rawData: [String: String]
    let validationErrors: [String]

    enum CodingKeys: String, CodingKey {
        case rowIndex = "row_index"
        case rawData = "raw_data"
        case validationErrors = "validation_errors"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        rowIndex = try c.decode(Int64.self, forKey: .rowIndex)
        validationErrors = try c.decodeIfPresent([String].self, forKey: .validationErrors) ?? []
        let rawMap = try c.decodeIfPresent([String: FlexibleJSONValue].self, forKey: .rawData) ?? [:]
        var flattened: [String: String] = [:]
        flattened.reserveCapacity(rawMap.count)
        for (k, v) in rawMap {
            flattened[k] = v.text
        }
        rawData = flattened
    }
}

struct MigrationSessionRowsResponse: Decodable {
    let sessionId: String
    let totalRows: Int64
    let rows: [MigrationStagedRow]

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case totalRows = "total_rows"
        case rows
    }
}

struct MigrationDryRunConflict: Decodable, Identifiable {
    var id: String { "\(sessionId):\(rowIndex):\(code):\(field ?? "")" }
    let sessionId: String
    let pack: String
    let rowIndex: Int64
    let code: String
    let severity: String
    let field: String?
    let message: String
    let suggestion: String?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case pack
        case rowIndex = "row_index"
        case code, severity, field, message, suggestion
    }
}

struct MigrationPackDryRunReport: Decodable, Identifiable {
    var id: String { sessionId }
    let sessionId: String
    let pack: String
    let rowsTotal: Int64
    let rowsValid: Int64
    let rowsInvalid: Int64
    let wouldCreate: Int64
    let wouldUpdate: Int64
    let wouldSkip: Int64
    let moneyImpactMinor: Int64
    let currency: String
    let blockingCount: Int64
    let warningCount: Int64
    let coldLotWithoutLive: Int64?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case pack
        case rowsTotal = "rows_total"
        case rowsValid = "rows_valid"
        case rowsInvalid = "rows_invalid"
        case wouldCreate = "would_create"
        case wouldUpdate = "would_update"
        case wouldSkip = "would_skip"
        case moneyImpactMinor = "money_impact_minor"
        case currency
        case blockingCount = "blocking_count"
        case warningCount = "warning_count"
        case coldLotWithoutLive = "cold_lot_migrated_without_live_baseline"
    }
}

struct MigrationDryRunSummary: Decodable {
    let projectId: String
    let contentFingerprint: String
    let packs: [MigrationPackDryRunReport]
    let conflicts: [MigrationDryRunConflict]?
    let balancedJournalCheck: Bool
    let journalDebitMinor: Int64
    let journalCreditMinor: Int64
    let openingBalanceAccount: String
    let readyToApprove: Bool
    let coldLotWarningCount: Int64
    let onHandBelowReservedHits: Int64
    let evaluatedAt: String
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case projectId = "project_id"
        case contentFingerprint = "content_fingerprint"
        case packs, conflicts
        case balancedJournalCheck = "balanced_journal_check"
        case journalDebitMinor = "journal_debit_minor"
        case journalCreditMinor = "journal_credit_minor"
        case openingBalanceAccount = "opening_balance_account"
        case readyToApprove = "ready_to_approve"
        case coldLotWarningCount = "cold_lot_migrated_without_live_baseline"
        case onHandBelowReservedHits = "on_hand_below_reserved"
        case evaluatedAt = "evaluated_at"
        case expiresAt = "expires_at"
    }
}

struct MigrationReport: Decodable, Identifiable {
    var id: String { reportId }
    let reportId: String
    let projectId: String?
    let sessionId: String?
    let reportKind: String
    let rowsTotal: Int64
    let rowsValid: Int64
    let rowsInvalid: Int64
    let wouldCreate: Int64
    let wouldUpdate: Int64
    let wouldSkip: Int64
    let moneyImpactMinor: Int64
    let currency: String
    let blockingCount: Int64
    let warningCount: Int64

    enum CodingKeys: String, CodingKey {
        case reportId = "report_id"
        case projectId = "project_id"
        case sessionId = "session_id"
        case reportKind = "report_kind"
        case rowsTotal = "rows_total"
        case rowsValid = "rows_valid"
        case rowsInvalid = "rows_invalid"
        case wouldCreate = "would_create"
        case wouldUpdate = "would_update"
        case wouldSkip = "would_skip"
        case moneyImpactMinor = "money_impact_minor"
        case currency
        case blockingCount = "blocking_count"
        case warningCount = "warning_count"
    }
}

struct MigrationDryRunResponse: Decodable {
    let report: MigrationReport
    let summary: MigrationDryRunSummary
}

struct MigrationApproveRequest: Encodable {
    let applyMode: String?
    let setOnHandConfirmed: Bool?
    let expectedReportId: String?

    enum CodingKeys: String, CodingKey {
        case applyMode = "apply_mode"
        case setOnHandConfirmed = "set_on_hand_confirmed"
        case expectedReportId = "expected_report_id"
    }
}

struct MigrationApproveResponse: Decodable {
    let session: MigrationSession
}

struct MigrationApplyWarning: Decodable, Identifiable {
    var id: String { "\(rowIndex):\(code)" }
    let rowIndex: Int64
    let code: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case rowIndex = "row_index"
        case code, message
    }
}

struct MigrationApplyResult: Decodable {
    let session: MigrationSession
    let alreadyApplied: Bool?
    let created: Int64
    let updated: Int64
    let skipped: Int64
    let chunksApplied: Int64
    let lastRowIndex: Int64
    let warnings: [MigrationApplyWarning]?

    enum CodingKeys: String, CodingKey {
        case session
        case alreadyApplied = "already_applied"
        case created, updated, skipped
        case chunksApplied = "chunks_applied"
        case lastRowIndex = "last_row_index"
        case warnings
    }
}

struct MigrationRollbackResult: Decodable {
    let session: MigrationSession
    let revertedSnapshots: Int64
    let reversedInvoices: Int64
    let supersededOrders: Int64
    let reversedAmountMinor: Int64

    enum CodingKeys: String, CodingKey {
        case session
        case revertedSnapshots = "reverted_snapshots"
        case reversedInvoices = "reversed_invoices"
        case supersededOrders = "superseded_orders"
        case reversedAmountMinor = "reversed_amount_minor"
    }
}

struct MigrationOpenOrderReleaseRequest: Encodable {
    let orderIds: [String]
    let targetStatus: String

    enum CodingKeys: String, CodingKey {
        case orderIds = "order_ids"
        case targetStatus = "target_status"
    }
}

struct MigrationOpenOrderReleaseResponse: Decodable {
    let releasedOrderIds: [String]
    let skippedOrderIds: [String]?
    let errors: [String: String]?

    enum CodingKeys: String, CodingKey {
        case releasedOrderIds = "released_order_ids"
        case skippedOrderIds = "skipped_order_ids"
        case errors
    }
}

struct MigrationCutoverChecklistItem: Decodable, Identifiable {
    var id: String { name }
    let name: String
    let passed: Bool
    let detail: String
}

struct MigrationCutoverResult: Decodable {
    let project: MigrationProject
    let sessions: [MigrationSession]
    let checklist: [MigrationCutoverChecklistItem]
    let cutoverAt: String

    enum CodingKeys: String, CodingKey {
        case project, sessions, checklist
        case cutoverAt = "cutover_at"
    }
}

// MARK: - INN Lookup & Join Requests

struct INNLookupRequest: Encodable {
    let inn: String
    let tenantType: String

    enum CodingKeys: String, CodingKey {
        case inn
        case tenantType = "tenant_type"
    }
}

struct INNLookupResponse: Decodable, Equatable {
    let status: String
    let normalizedInn: String
    let entityType: String?
    let legalName: String?
    let address: String?
    let registryState: String?
    let cached: Bool
    let existingTenantId: String?
    let existingTenantType: String?
    let existingLegalName: String?
    let existingJoinEligible: Bool

    enum CodingKeys: String, CodingKey {
        case status
        case normalizedInn = "normalized_inn"
        case entityType = "entity_type"
        case legalName = "legal_name"
        case address
        case registryState = "registry_state"
        case cached
        case existingTenantId = "existing_tenant_id"
        case existingTenantType = "existing_tenant_type"
        case existingLegalName = "existing_legal_name"
        case existingJoinEligible = "existing_join_eligible"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        status = try c.decode(String.self, forKey: .status)
        normalizedInn = try c.decodeIfPresent(String.self, forKey: .normalizedInn) ?? ""
        entityType = try c.decodeIfPresent(String.self, forKey: .entityType)
        legalName = try c.decodeIfPresent(String.self, forKey: .legalName)
        address = try c.decodeIfPresent(String.self, forKey: .address)
        registryState = try c.decodeIfPresent(String.self, forKey: .registryState)
        cached = try c.decodeIfPresent(Bool.self, forKey: .cached) ?? false
        existingTenantId = try c.decodeIfPresent(String.self, forKey: .existingTenantId)
        existingTenantType = try c.decodeIfPresent(String.self, forKey: .existingTenantType)
        existingLegalName = try c.decodeIfPresent(String.self, forKey: .existingLegalName)
        existingJoinEligible = try c.decodeIfPresent(Bool.self, forKey: .existingJoinEligible) ?? false
    }
}

struct JoinRequestConflictPayload: Decodable, Equatable, Sendable {
    let error: String
    let requestId: String?
    let targetTenantId: String?
    let legalName: String?
    let normalizedInn: String?

    enum CodingKeys: String, CodingKey {
        case error
        case requestId = "request_id"
        case targetTenantId = "target_tenant_id"
        case legalName = "legal_name"
        case normalizedInn = "normalized_inn"
    }
}

struct JoinRequestItem: Decodable, Identifiable, Equatable {
    var id: String { requestId }
    let requestId: String
    let homeCell: String
    let targetTenantType: String
    let targetTenantId: String
    let normalizedInn: String
    let requesterPhone: String
    let requesterName: String?
    let requesterEmail: String?
    let roleSought: String
    let status: String
    let decidedBy: String?
    let decidedAt: String?
    let decisionReason: String?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case requestId = "request_id"
        case homeCell = "home_cell"
        case targetTenantType = "target_tenant_type"
        case targetTenantId = "target_tenant_id"
        case normalizedInn = "normalized_inn"
        case requesterPhone = "requester_phone"
        case requesterName = "requester_name"
        case requesterEmail = "requester_email"
        case roleSought = "role_sought"
        case status
        case decidedBy = "decided_by"
        case decidedAt = "decided_at"
        case decisionReason = "decision_reason"
        case createdAt = "created_at"
    }
}

struct JoinRequestsResponse: Decodable {
    let supplierId: String?
    let requests: [JoinRequestItem]

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case requests
    }
}

// MARK: - Supplier AR / Nasiya Opening Balances

struct SupplierARInvoice: Decodable, Identifiable, Equatable {
    var id: String { invoiceId }
    let invoiceId: String
    let supplierId: String
    let retailerId: String
    let orderId: String
    let status: String
    let principalMinor: Int64
    let balanceMinor: Int64
    let currency: String
    let dueAt: String?
    let source: String?
    let externalId: String?
    let acknowledgedAt: String?
    let disputedAt: String?
    let disputeReason: String?

    enum CodingKeys: String, CodingKey {
        case invoiceId = "invoice_id"
        case supplierId = "supplier_id"
        case retailerId = "retailer_id"
        case orderId = "order_id"
        case status
        case principalMinor = "principal_minor"
        case balanceMinor = "balance_minor"
        case currency
        case dueAt = "due_at"
        case source
        case externalId = "external_id"
        case acknowledgedAt = "acknowledged_at"
        case disputedAt = "disputed_at"
        case disputeReason = "dispute_reason"
    }

    var isMigrationOrigin: Bool {
        (source ?? "").uppercased() == "MIGRATION"
    }

    var ackState: String {
        if disputedAt != nil { return "DISPUTED" }
        if acknowledgedAt != nil { return "ACKNOWLEDGED" }
        return "PENDING_ACK"
    }
}

struct SupplierARInvoicesResponse: Decodable {
    let invoices: [SupplierARInvoice]
}

struct MigrationApplyProgressEvent: Equatable {
    let sessionId: String
    let processedRows: Int64
    let totalRows: Int64
    let chunkId: String?
}
