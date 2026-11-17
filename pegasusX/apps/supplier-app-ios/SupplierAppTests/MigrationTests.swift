import XCTest
@testable import SupplierAppIOS

final class MigrationTests: XCTestCase {
    override func tearDown() {
        MigrationURLProtocolStub.reset()
        APIClient.shared.setTestSession(URLSession(configuration: .ephemeral))
        super.tearDown()
    }

    func testSHA256HexMatchesKnownVector() {
        let digest = MigrationService.sha256Hex(of: Data("abc".utf8))
        XCTAssertEqual(digest, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
    }

    func testDirectSignedPUTUploadSendsBinaryPayload() async throws {
        installStub { request, body in
            XCTAssertEqual(request.httpMethod, "PUT")
            XCTAssertEqual(request.value(forHTTPHeaderField: "Content-Type"), "text/csv")
            XCTAssertEqual(String(data: body, encoding: .utf8), "sku,qty\nA1,5\n")
            return (200, [:], Data())
        }

        let ticket = try makeUploadTicket(
            url: "https://storage.googleapis.com/bucket/direct.csv?sig=1",
            resumable: false
        )
        try await MigrationService.uploadArtifactToSignedTicket(
            ticket: ticket,
            data: Data("sku,qty\nA1,5\n".utf8)
        )
        XCTAssertEqual(MigrationURLProtocolStub.recordedRequests.count, 1)
    }

    func testResumableGCSUploadChunksOver8MiBAndResumesWith308And200() async throws {
        let totalSize = 9 * 1024 * 1024 // 9 MiB -> 2 chunks (8 MiB + 1 MiB)
        let payload = Data(repeating: 0x41, count: totalSize)
        var capturedChunks = 0

        installStub { request, body in
            if request.httpMethod == "POST" {
                XCTAssertEqual(request.value(forHTTPHeaderField: "x-goog-resumable"), "start")
                return (
                    200,
                    ["Location": "https://storage.googleapis.com/upload/session-123"],
                    Data()
                )
            }
            capturedChunks += 1
            XCTAssertEqual(request.url?.absoluteString, "https://storage.googleapis.com/upload/session-123")
            let range = request.value(forHTTPHeaderField: "Content-Range")
            if capturedChunks == 1 {
                XCTAssertEqual(range, "bytes 0-8388607/\(totalSize)")
                XCTAssertEqual(body.count, 8 * 1024 * 1024)
                return (308, [:], Data())
            } else {
                XCTAssertEqual(range, "bytes 8388608-\(totalSize - 1)/\(totalSize)")
                XCTAssertEqual(body.count, 1 * 1024 * 1024)
                return (200, [:], Data())
            }
        }

        let ticket = try makeUploadTicket(
            url: "https://storage.googleapis.com/upload/start?sig=1",
            resumable: true
        )
        try await MigrationService.uploadArtifactToSignedTicket(ticket: ticket, data: payload)
        XCTAssertEqual(capturedChunks, 2)
    }

    @MainActor
    func testOnboardingFailsClosedWhenINNRegistryUnavailable() async {
        var setupEndpointCalled = false
        installStub { request, _ in
            if request.url?.path.contains("inn/lookup") == true {
                let json = """
                {"status":"UNAVAILABLE","normalized_inn":"301234567","cached":false,"existing_join_eligible":false}
                """
                return (200, ["Content-Type": "application/json"], Data(json.utf8))
            }
            setupEndpointCalled = true
            return (200, ["Content-Type": "application/json"], Data("{}".utf8))
        }

        let vm = OnboardingViewModel()
        vm.taxId = "301234567"
        vm.headquartersAddress = "10 Amir Temur"
        vm.city = "Tashkent"

        let ok = await vm.setupBusiness(tokenStore: TokenStore.shared)
        XCTAssertFalse(ok)
        XCTAssertFalse(setupEndpointCalled)
        XCTAssertEqual(vm.innLookup?.status, "UNAVAILABLE")
        XCTAssertNotNil(vm.error)
    }

    @MainActor
    func testOnboardingDecodes409JoinRequestPendingConflict() async {
        installStub { request, _ in
            if request.url?.path.contains("inn/lookup") == true {
                let json = """
                {
                  "status":"VERIFIED",
                  "normalized_inn":"301234567",
                  "legal_name":"OOO Samarkand Trade",
                  "address":"10 Amir Temur",
                  "cached":false,
                  "existing_tenant_id":"sup_existing_1",
                  "existing_tenant_type":"SUPPLIER",
                  "existing_legal_name":"OOO Samarkand Trade",
                  "existing_join_eligible":true
                }
                """
                return (200, ["Content-Type": "application/json"], Data(json.utf8))
            }
            let conflict = """
            {
              "error":"join_request_pending",
              "request_id":"jr_99",
              "target_tenant_id":"sup_existing_1",
              "legal_name":"OOO Samarkand Trade",
              "normalized_inn":"301234567"
            }
            """
            return (409, ["Content-Type": "application/json"], Data(conflict.utf8))
        }

        let vm = OnboardingViewModel()
        vm.taxId = "301234567"
        vm.headquartersAddress = "10 Amir Temur"
        vm.city = "Tashkent"

        let ok = await vm.setupBusiness(tokenStore: TokenStore.shared)
        XCTAssertFalse(ok)
        XCTAssertTrue(vm.joinRequestSent)
        XCTAssertEqual(vm.joinRequestConflict?.requestId, "jr_99")
        XCTAssertEqual(vm.joinRequestConflict?.targetTenantId, "sup_existing_1")
        XCTAssertNil(vm.error)
    }

    @MainActor
    func testRealtimeHubTracksMigrationApplyProgressFlatAndNested() throws {
        let hub = SupplierRealtimeHub()

        let flatJSON = Data("""
        {"type":"MIGRATION_APPLY_PROGRESS","session_id":"sess_1","processed_rows":500,"total_rows":1200}
        """.utf8)
        let flatEvent = try JSONDecoder().decode(SupplierLiveEvent.self, from: flatJSON)
        hub.bump(event: flatEvent)
        XCTAssertEqual(hub.migrationProgressBySession["sess_1"]?.processedRows, 500)
        XCTAssertEqual(hub.migrationProgressBySession["sess_1"]?.totalRows, 1200)

        let nestedJSON = Data("""
        {"type":"MIGRATION_APPLY_PROGRESS","payload":{"session_id":"sess_1","processed_rows":1200,"total_rows":1200}}
        """.utf8)
        let nestedEvent = try JSONDecoder().decode(SupplierLiveEvent.self, from: nestedJSON)
        hub.bump(event: nestedEvent)
        XCTAssertEqual(hub.migrationProgressBySession["sess_1"]?.processedRows, 1200)
    }

    func testStagedRowAndIdempotencyKeysAreDeterministic() throws {
        let rowJSON = Data("""
        {"row_index":3,"raw_data":{"sku":"SKU-1","qty":40,"active":true,"note":null},"validation_errors":[]}
        """.utf8)
        let row = try JSONDecoder().decode(MigrationStagedRow.self, from: rowJSON)
        XCTAssertEqual(row.rawData["sku"], "SKU-1")
        XCTAssertEqual(row.rawData["qty"], "40")
        XCTAssertEqual(row.rawData["active"], "true")
        XCTAssertEqual(row.rawData["note"], "")

        let k1 = SupplierIdempotencyKeys.migrationSessionApply(sessionId: "s1", fingerprint: "fp1")
        let k2 = SupplierIdempotencyKeys.migrationSessionApply(sessionId: "s1", fingerprint: "fp1")
        let k3 = SupplierIdempotencyKeys.migrationSessionApply(sessionId: "s1", fingerprint: "fp2")
        XCTAssertEqual(k1, k2)
        XCTAssertNotEqual(k1, k3)
    }

    private func makeUploadTicket(url: String, resumable: Bool) throws -> MigrationUploadTicket {
        let json = """
        {
          "artifact_id":"art_1",
          "file_name":"goods.csv",
          "gcs_path":"gs://b/goods.csv",
          "upload_url":"\(url)",
          "method":"PUT",
          "content_type":"text/csv",
          "resumable":\(resumable ? "true" : "false"),
          "expires_at":"2026-12-01T00:00:00Z"
        }
        """
        return try JSONDecoder().decode(MigrationUploadTicket.self, from: Data(json.utf8))
    }

    private func installStub(handler: @escaping (URLRequest, Data) -> (Int, [String: String], Data)) {
        MigrationURLProtocolStub.handler = handler
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [MigrationURLProtocolStub.self]
        APIClient.shared.setTestSession(URLSession(configuration: config))
    }
}

private final class MigrationURLProtocolStub: URLProtocol {
    nonisolated(unsafe) static var handler: ((URLRequest, Data) -> (Int, [String: String], Data))?
    nonisolated(unsafe) static var recordedRequests: [URLRequest] = []

    static func reset() {
        handler = nil
        recordedRequests = []
    }

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        let bodyData = request.httpBody ?? Self.readBodyStream(request.httpBodyStream)
        Self.recordedRequests.append(request)

        let url = request.url ?? SupplierURL.devBootstrap
        let (status, headers, responseData) = Self.handler?(request, bodyData) ?? (200, [:], Data())
        guard let response = HTTPURLResponse(
            url: url,
            statusCode: status,
            httpVersion: nil,
            headerFields: headers
        ) else {
            client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
            return
        }
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: responseData)
        client?.urlProtocolDidFinishLoading(self)
    }

    override func stopLoading() {}

    private static func readBodyStream(_ stream: InputStream?) -> Data {
        guard let stream else { return Data() }
        stream.open()
        defer { stream.close() }
        var data = Data()
        let bufferSize = 65_536
        var buffer = [UInt8](repeating: 0, count: bufferSize)
        while stream.hasBytesAvailable {
            let read = stream.read(&buffer, maxLength: bufferSize)
            if read > 0 {
                data.append(buffer, count: read)
            } else {
                break
            }
        }
        return data
    }
}
