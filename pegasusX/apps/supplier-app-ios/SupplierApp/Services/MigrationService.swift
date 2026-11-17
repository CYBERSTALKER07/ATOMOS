import CryptoKit
import Foundation

enum MigrationService {
    private static let resumableChunkSize = 8 * 1024 * 1024 // 8 MiB aligned to GCS 256 KiB multiple

    static func sha256Hex(of data: Data) -> String {
        let digest = SHA256.hash(data: data)
        return digest.map { String(format: "%02x", $0) }.joined()
    }

    static func listProjects() async throws -> [MigrationProject] {
        let resp: MigrationProjectsResponse = try await APIClient.shared.get("v1/supplier/migrations/projects")
        return resp.projects
    }

    static func createProject(
        name: String,
        defaultWarehouseId: String?,
        idempotencyKey: String
    ) async throws -> MigrationProject {
        struct Body: Encodable {
            let name: String
            let defaultWarehouseId: String?
            enum CodingKeys: String, CodingKey {
                case name
                case defaultWarehouseId = "default_warehouse_id"
            }
        }
        return try await APIClient.shared.post(
            "v1/supplier/migrations/projects",
            body: Body(
                name: name,
                defaultWarehouseId: defaultWarehouseId?.isEmpty == false ? defaultWarehouseId : nil
            ),
            idempotencyKey: idempotencyKey
        )
    }

    static func getProjectTree(projectId: String) async throws -> MigrationProjectTree {
        try await APIClient.shared.get("v1/supplier/migrations/projects/\(projectId)")
    }

    static func createBundle(
        projectId: String,
        files: [MigrationDeclareFile],
        idempotencyKey: String
    ) async throws -> MigrationCreateBundleResponse {
        try await APIClient.shared.post(
            "v1/supplier/migrations/projects/\(projectId)/bundles",
            body: MigrationCreateBundleRequest(files: files),
            idempotencyKey: idempotencyKey
        )
    }

    static func uploadArtifactToSignedTicket(
        ticket: MigrationUploadTicket,
        data: Data,
        onProgress: (@MainActor (Int64, Int64) -> Void)? = nil
    ) async throws {
        let total = Int64(data.count)
        if ticket.uploadUrl.hasPrefix("https://local.invalid/") {
            await onProgress?(total, total)
            return
        }
        guard let url = URL(string: ticket.uploadUrl) else {
            throw APIError.invalidURL
        }

        var headers = ticket.requiredHeaders ?? [:]
        if headers["Content-Type"] == nil && !ticket.contentType.isEmpty {
            headers["Content-Type"] = ticket.contentType
        }

        if !ticket.resumable || data.count <= resumableChunkSize {
            let method = ticket.method == "POST" ? "PUT" : (ticket.method.isEmpty ? "PUT" : ticket.method)
            let http = try await APIClient.shared.uploadBinary(to: url, method: method, headers: headers, body: data)
            guard (200...299).contains(http.statusCode) else {
                throw APIError.httpError(http.statusCode)
            }
            await onProgress?(total, total)
            return
        }

        var initHeaders = headers
        initHeaders["x-goog-resumable"] = "start"
        let startResp = try await APIClient.shared.uploadBinary(to: url, method: "POST", headers: initHeaders, body: Data())
        guard (200...299).contains(startResp.statusCode) else {
            throw APIError.httpError(startResp.statusCode)
        }
        let location = startResp.value(forHTTPHeaderField: "Location")
        let sessionURL = location.flatMap(URL.init(string:)) ?? url

        var offset = 0
        while offset < data.count {
            let end = min(offset + resumableChunkSize, data.count)
            let chunk = data.subdata(in: offset..<end)
            let rangeHeader = "bytes \(offset)-\(end - 1)/\(data.count)"
            let chunkResp = try await APIClient.shared.uploadBinary(
                to: sessionURL,
                method: "PUT",
                headers: ["Content-Range": rangeHeader],
                body: chunk
            )
            guard (200...299).contains(chunkResp.statusCode) || chunkResp.statusCode == 308 else {
                throw APIError.httpError(chunkResp.statusCode)
            }
            offset = end
            await onProgress?(Int64(offset), total)
        }
    }

    static func markBundleUploaded(
        projectId: String,
        bundleId: String,
        artifacts: [String: MigrationUploadedArtifactMeta],
        idempotencyKey: String,
        sync: Bool = true
    ) async throws -> MigrationBundleUploadedResponse {
        try await APIClient.shared.post(
            "v1/supplier/migrations/projects/\(projectId)/bundles/\(bundleId)/uploaded",
            body: MigrationBundleUploadedRequest(artifacts: artifacts),
            query: sync ? ["sync": "true"] : [:],
            idempotencyKey: idempotencyKey
        )
    }

    static func overrideBundleSort(
        projectId: String,
        bundleId: String,
        request: MigrationBundleSortOverrideRequest,
        idempotencyKey: String
    ) async throws -> MigrationBundleSortOverrideResponse {
        try await APIClient.shared.patch(
            "v1/supplier/migrations/projects/\(projectId)/bundles/\(bundleId)/sort",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func getSession(sessionId: String) async throws -> MigrationSession {
        try await APIClient.shared.get("v1/supplier/migrations/sessions/\(sessionId)")
    }

    static func getSessionRows(
        sessionId: String,
        limit: Int = 20,
        offset: Int = 0,
        invalidOnly: Bool = false
    ) async throws -> MigrationSessionRowsResponse {
        var query: [String: String] = [
            "limit": String(limit),
            "offset": String(offset),
        ]
        if invalidOnly {
            query["invalid_only"] = "1"
        }
        return try await APIClient.shared.get("v1/supplier/migrations/sessions/\(sessionId)/rows", query: query)
    }

    static func getSessionMapping(sessionId: String) async throws -> MigrationGetSessionMappingResponse {
        try await APIClient.shared.get("v1/supplier/migrations/sessions/\(sessionId)/mapping")
    }

    static func confirmSessionMapping(
        sessionId: String,
        request: MigrationConfirmMappingRequest,
        idempotencyKey: String
    ) async throws -> MigrationConfirmMappingResponse {
        try await APIClient.shared.post(
            "v1/supplier/migrations/sessions/\(sessionId)/mapping",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func runProjectDryRun(
        projectId: String,
        applyMode: String,
        idempotencyKey: String
    ) async throws -> MigrationDryRunResponse {
        struct Body: Encodable {
            let applyMode: String
            enum CodingKeys: String, CodingKey { case applyMode = "apply_mode" }
        }
        return try await APIClient.shared.post(
            "v1/supplier/migrations/projects/\(projectId)/dry-run",
            body: Body(applyMode: applyMode),
            idempotencyKey: idempotencyKey
        )
    }

    static func runSessionDryRun(
        sessionId: String,
        applyMode: String,
        idempotencyKey: String
    ) async throws -> MigrationDryRunResponse {
        struct Body: Encodable {
            let applyMode: String
            enum CodingKeys: String, CodingKey { case applyMode = "apply_mode" }
        }
        return try await APIClient.shared.post(
            "v1/supplier/migrations/sessions/\(sessionId)/dry-run",
            body: Body(applyMode: applyMode),
            idempotencyKey: idempotencyKey
        )
    }

    static func getReport(reportId: String) async throws -> MigrationDryRunResponse {
        try await APIClient.shared.get("v1/supplier/migrations/reports/\(reportId)")
    }

    static func approveSession(
        sessionId: String,
        request: MigrationApproveRequest,
        idempotencyKey: String
    ) async throws -> MigrationApproveResponse {
        try await APIClient.shared.post(
            "v1/supplier/migrations/sessions/\(sessionId)/approve",
            body: request,
            idempotencyKey: idempotencyKey
        )
    }

    static func applySession(
        sessionId: String,
        applyMode: String,
        chunkSize: Int = 500,
        idempotencyKey: String
    ) async throws -> MigrationApplyResult {
        struct Body: Encodable {
            let applyMode: String
            let chunkSize: Int
            enum CodingKeys: String, CodingKey {
                case applyMode = "apply_mode"
                case chunkSize = "chunk_size"
            }
        }
        return try await APIClient.shared.post(
            "v1/supplier/migrations/sessions/\(sessionId)/apply",
            body: Body(applyMode: applyMode, chunkSize: chunkSize),
            idempotencyKey: idempotencyKey
        )
    }

    static func rollbackSession(sessionId: String, idempotencyKey: String) async throws -> MigrationRollbackResult {
        try await APIClient.shared.post(
            "v1/supplier/migrations/sessions/\(sessionId)/rollback",
            body: [String: String](),
            idempotencyKey: idempotencyKey
        )
    }

    static func releaseOpenOrders(
        projectId: String,
        orderIds: [String],
        targetStatus: String,
        idempotencyKey: String
    ) async throws -> MigrationOpenOrderReleaseResponse {
        try await APIClient.shared.post(
            "v1/supplier/migrations/projects/\(projectId)/open-orders/release",
            body: MigrationOpenOrderReleaseRequest(orderIds: orderIds, targetStatus: targetStatus),
            idempotencyKey: idempotencyKey
        )
    }

    static func cutoverProject(projectId: String, idempotencyKey: String) async throws -> MigrationCutoverResult {
        try await APIClient.shared.post(
            "v1/supplier/migrations/projects/\(projectId)/cutover",
            body: [String: String](),
            idempotencyKey: idempotencyKey
        )
    }

    // MARK: - INN Lookup & Join Requests

    static func lookupINN(inn: String, tenantType: String = "SUPPLIER") async throws -> INNLookupResponse {
        try await APIClient.shared.post(
            "v1/platform/inn/lookup",
            body: INNLookupRequest(inn: inn, tenantType: tenantType),
            authenticated: true
        )
    }

    static func listJoinRequests(status: String = "PENDING", limit: Int = 100) async throws -> [JoinRequestItem] {
        var query: [String: String] = ["limit": String(limit)]
        if !status.isEmpty {
            query["status"] = status
        }
        let resp: JoinRequestsResponse = try await APIClient.shared.get("v1/supplier/join-requests", query: query)
        return resp.requests
    }

    static func approveJoinRequest(requestId: String, reason: String, idempotencyKey: String) async throws {
        struct Body: Encodable { let reason: String }
        try await APIClient.shared.postVoid(
            "v1/supplier/join-requests/\(requestId)/approve",
            body: Body(reason: reason),
            idempotencyKey: idempotencyKey
        )
    }

    static func rejectJoinRequest(requestId: String, reason: String, idempotencyKey: String) async throws {
        struct Body: Encodable { let reason: String }
        try await APIClient.shared.postVoid(
            "v1/supplier/join-requests/\(requestId)/reject",
            body: Body(reason: reason),
            idempotencyKey: idempotencyKey
        )
    }

    // MARK: - Supplier AR / Nasiya Opening Balances

    static func listSupplierARInvoices(status: String? = nil, limit: Int = 100) async throws -> [SupplierARInvoice] {
        var query: [String: String] = ["limit": String(limit)]
        if let status, !status.isEmpty {
            query["status"] = status
        }
        let resp: SupplierARInvoicesResponse = try await APIClient.shared.get("v1/supplier/ar/invoices", query: query)
        return resp.invoices
    }
}
