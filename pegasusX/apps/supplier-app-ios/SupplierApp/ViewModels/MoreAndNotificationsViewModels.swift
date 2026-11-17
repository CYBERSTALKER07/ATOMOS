import Foundation
import Observation

struct CreditAdminDisableBody: Encodable, Sendable {
    let ticket_id: String
    let reason: String
}

struct CreditAdminDisableResponse: Decodable, Sendable {
    let status: String?
    let error: String?
}

@Observable
@MainActor
final class CreditAdminDisableViewModel {
    var mode = "relationship"
    var supplierId = ""
    var retailerId = ""
    var ticketId = ""
    var reason = ""
    var message: String?
    var busy = false

    private let submitter: @Sendable (String, CreditAdminDisableBody, String) async throws -> CreditAdminDisableResponse

    init(
        submitter: @escaping @Sendable (String, CreditAdminDisableBody, String) async throws -> CreditAdminDisableResponse = { path, body, key in
            try await APIClient.shared.post(path, body: body, idempotencyKey: key)
        }
    ) {
        self.submitter = submitter
    }

    var isProgramMode: Bool {
        mode == "mobile_supplier.chrome.source_program"
    }

    var canSubmit: Bool {
        let sup = supplierId.trimmingCharacters(in: .whitespacesAndNewlines)
        let ret = retailerId.trimmingCharacters(in: .whitespacesAndNewlines)
        let tck = ticketId.trimmingCharacters(in: .whitespacesAndNewlines)
        let rsn = reason.trimmingCharacters(in: .whitespacesAndNewlines)
        return !busy && !sup.isEmpty && !tck.isEmpty && !rsn.isEmpty && (isProgramMode || !ret.isEmpty)
    }

    var targetPath: String {
        let sup = supplierId.trimmingCharacters(in: .whitespacesAndNewlines)
        let ret = retailerId.trimmingCharacters(in: .whitespacesAndNewlines)
        return isProgramMode
            ? "v1/admin/credit-program/\(sup)/disable"
            : "v1/admin/credit-relationships/\(sup)/\(ret)/disable"
    }

    func submit() async {
        guard canSubmit else { return }
        busy = true
        defer { busy = false }
        do {
            let sup = supplierId.trimmingCharacters(in: .whitespacesAndNewlines)
            let ret = retailerId.trimmingCharacters(in: .whitespacesAndNewlines)
            let tck = ticketId.trimmingCharacters(in: .whitespacesAndNewlines)
            let rsn = reason.trimmingCharacters(in: .whitespacesAndNewlines)
            let body = CreditAdminDisableBody(ticket_id: tck, reason: rsn)
            let path = targetPath
            let key = SupplierIdempotencyKeys.creditAdminDisable(
                mode: mode,
                supplierId: sup,
                retailerId: isProgramMode ? "" : ret,
                ticketId: tck
            )
            let resp = try await submitter(path, body, key)
            message = ActionResultText.nonEmpty(status: resp.status, error: resp.error)
            PegasusHaptics.success()
        } catch {
            PegasusHaptics.error()
            message = error.localizedDescription
        }
    }
}

@Observable
@MainActor
final class JSONFeedViewModel {
    var bodyText = ""
    var errorText: String?
    var loading = true
    private var loadGeneration = 0
    private let fetcher: @Sendable (String, [String: String]) async throws -> String

    init(
        fetcher: @escaping @Sendable (String, [String: String]) async throws -> String = { path, query in
            try await APIClient.shared.getJSONString(path, query: query)
        }
    ) {
        self.fetcher = fetcher
    }

    func load(path: String, query: [String: String], silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !bodyText.isEmpty || (errorText == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &errorText)
        do {
            let loaded = try await fetcher(path, query)
            guard generation == loadGeneration else { return }
            bodyText = loaded
            SupplierScreenLoad.succeed(error: &errorText)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &errorText,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

@Observable
@MainActor
final class PlaybooksViewModel {
    var rows: [ControlTowerPlaybook] = []
    var errorText: String?
    var loading = true
    private var loadGeneration = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &errorText)
        do {
            let loadedRows = try await SupplierOperationsService.playbooks()
            guard generation == loadGeneration else { return }
            rows = loadedRows
            errorText = nil
            SupplierScreenLoad.succeed(error: &errorText)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &errorText,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

@Observable
@MainActor
final class ScoredExceptionsViewModel {
    var rows: [ScoredException] = []
    var errorText: String?
    var loading = true
    private var loadGeneration = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &errorText)
        do {
            let loadedRows = try await SupplierOperationsService.scoredExceptions()
            guard generation == loadGeneration else { return }
            rows = loadedRows
            errorText = nil
            SupplierScreenLoad.succeed(error: &errorText)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &errorText,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}

struct SupplierNotificationsPage: Decodable, Sendable {
    let notifications: [SupplierNotificationItem]
    let unreadCount: Int
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case notifications
        case unreadCount = "unread_count"
        case hasMore = "has_more"
    }
}

struct MarkNotificationsReadRequest: Encodable, Sendable {
    let markAll: Bool

    enum CodingKeys: String, CodingKey {
        case markAll = "mark_all"
    }
}

@Observable
@MainActor
final class NotificationInboxViewModel {
    var items: [SupplierNotificationItem] = []
    var unreadCount = 0
    var loading = true
    var error: String?
    private var loadGeneration = 0

    private let pageFetcher: @Sendable () async throws -> SupplierNotificationsPage
    private let markAllHandler: @Sendable (String) async throws -> Void

    init(
        pageFetcher: @escaping @Sendable () async throws -> SupplierNotificationsPage = {
            try await APIClient.shared.get(
                "v1/user/notifications",
                query: ["limit": "50", "offset": "0"]
            )
        },
        markAllHandler: @escaping @Sendable (String) async throws -> Void = { key in
            try await APIClient.shared.postVoid(
                "v1/user/notifications/read",
                body: MarkNotificationsReadRequest(markAll: true),
                idempotencyKey: key
            )
        }
    ) {
        self.pageFetcher = pageFetcher
        self.markAllHandler = markAllHandler
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !items.isEmpty || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let page = try await pageFetcher()
            guard generation == loadGeneration else { return }
            items = page.notifications
            unreadCount = page.unreadCount
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func markAllRead() async {
        let unreadIds = items
            .filter { ($0.readAt ?? "").isEmpty }
            .map(\.id)
            .sorted()
        let fingerprint = "\(unreadCount)|\(unreadIds.joined(separator: ","))"
        let key = SupplierIdempotencyKeys.notificationsMarkAll(
            scopeId: SupplierIdempotencyKeys.supplierScopeId(),
            payloadFingerprint: fingerprint
        )
        do {
            try await markAllHandler(key)
            PegasusHaptics.success()
            await load(silent: true)
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: !items.isEmpty, error: &self.error)
        }
    }
}
