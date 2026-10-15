import Foundation

@MainActor
public final class EnterpriseAPIClient: ObservableObject {
    public static let shared = EnterpriseAPIClient()

    @Published public var baseURL: String = "http://localhost:3000"
    @Published public var isOnline: Bool = true
    @Published public var isSyncing: Bool = false
    @Published public var lastSyncTimestamp: Date? = nil

    // In-memory fallback buffer
    private var fallbackStore: [EnterpriseRequest] = [
        EnterpriseRequest(
            name: "Alexander Sterling",
            email: "asterling@alpinelogistics.com",
            company: "Alpine Logistics Group",
            phone: "+1 (415) 880-1200",
            role: "Fleet / Dispatch Lead",
            fleet_size: "11–25 sites",
            inquiry_type: "demo",
            subject: "Regional Cold-Chain Dispatch Pilot",
            message: "Need automated cross-docking sync across 14 perishable transit hubs with offline telematics.",
            status: "new",
            priority: "urgent"
        ),
        EnterpriseRequest(
            name: "Mariya K.",
            email: "mariya@samarkand-hub.uz",
            company: "Tashkent Central Warehouse",
            role: "Warehouse Manager",
            fleet_size: "4–10 sites",
            inquiry_type: "pilot",
            subject: "Warehouse Cell Integration",
            message: "Looking to deploy Pegasus barcode scanning and bin replenishment protocols.",
            status: "in_review",
            priority: "normal"
        )
    ]

    private init() {}

    // MARK: - Fetch Requests
    public func fetchRequests(status: String? = nil, inquiryType: String? = nil) async -> [EnterpriseRequest] {
        isSyncing = true
        defer {
            isSyncing = false
            lastSyncTimestamp = Date()
        }

        var components = URLComponents(string: "\(baseURL)/api/admin/requests")
        var queryItems: [URLQueryItem] = []
        if let status = status, status != "all" {
            queryItems.append(URLQueryItem(name: "status", value: status))
        }
        if let inquiryType = inquiryType, inquiryType != "all" {
            queryItems.append(URLQueryItem(name: "inquiry_type", value: inquiryType))
        }
        if !queryItems.isEmpty {
            components?.queryItems = queryItems
        }

        guard let url = components?.url else {
            return filteredFallback(status: status, inquiryType: inquiryType)
        }

        do {
            var request = URLRequest(url: url)
            request.timeoutInterval = 4.0
            request.setValue("application/json", forHTTPHeaderField: "Accept")

            let (data, response) = try await URLSession.shared.data(for: request)
            guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                isOnline = false
                return filteredFallback(status: status, inquiryType: inquiryType)
            }

            let decoded = try JSONDecoder().decode(EnterpriseAPIResponse<[EnterpriseRequest]>.self, from: data)
            if let list = decoded.data {
                isOnline = true
                self.fallbackStore = list
                return list
            }
            return filteredFallback(status: status, inquiryType: inquiryType)
        } catch {
            isOnline = false
            return filteredFallback(status: status, inquiryType: inquiryType)
        }
    }

    // MARK: - Submit Request
    public func submitRequest(
        name: String,
        email: String,
        company: String?,
        role: String?,
        fleetSize: String?,
        inquiryType: String,
        message: String?
    ) async throws -> EnterpriseRequest {
        let payload: [String: Any?] = [
            "name": name,
            "email": email,
            "company": company,
            "role": role,
            "fleet_size": fleetSize,
            "inquiry_type": inquiryType,
            "message": message,
            "source_page": "desktop_macos"
        ]

        guard let url = URL(string: "\(baseURL)/api/enterprise/request") else {
            let record = EnterpriseRequest(
                name: name,
                email: email,
                company: company,
                role: role,
                fleet_size: fleetSize,
                inquiry_type: inquiryType,
                message: message
            )
            fallbackStore.insert(record, at: 0)
            return record
        }

        do {
            var request = URLRequest(url: url)
            request.httpMethod = "POST"
            request.timeoutInterval = 5.0
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: payload.compactMapValues { $0 })

            let (data, response) = try await URLSession.shared.data(for: request)
            if let http = response as? HTTPURLResponse, http.statusCode == 200 {
                let decoded = try JSONDecoder().decode(EnterpriseAPIResponse<EnterpriseRequest>.self, from: data)
                if let item = decoded.data {
                    fallbackStore.insert(item, at: 0)
                    return item
                }
            }
        } catch {
            // Local fallback
        }

        let record = EnterpriseRequest(
            name: name,
            email: email,
            company: company,
            role: role,
            fleet_size: fleetSize,
            inquiry_type: inquiryType,
            message: message
        )
        fallbackStore.insert(record, at: 0)
        return record
    }

    // MARK: - Send Client Reply
    public func sendClientReply(
        requestId: String,
        recipient: String,
        subject: String,
        message: String
    ) async throws -> Bool {
        guard let url = URL(string: "\(baseURL)/api/admin/reply") else {
            return false
        }

        let payload: [String: String] = [
            "request_id": requestId,
            "recipient": recipient,
            "subject": subject,
            "message": message
        ]

        do {
            var request = URLRequest(url: url)
            request.httpMethod = "POST"
            request.timeoutInterval = 6.0
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: payload)

            let (_, response) = try await URLSession.shared.data(for: request)
            if let http = response as? HTTPURLResponse, http.statusCode == 200 {
                // Update local fallback
                if let idx = fallbackStore.firstIndex(where: { $0.id == requestId }) {
                    fallbackStore[idx].status = "contacted"
                    let note = "[Answered at \(Date())]: \(subject)\n\(message)"
                    fallbackStore[idx].admin_notes = note
                }
                return true
            }
        } catch {
            // Local update on offline fallback
            if let idx = fallbackStore.firstIndex(where: { $0.id == requestId }) {
                fallbackStore[idx].status = "contacted"
                fallbackStore[idx].admin_notes = "[Drafted Offline]: \(subject)\n\(message)"
            }
            return true
        }
        return false
    }

    // MARK: - Update Status
    public func updateStatus(id: String, status: String, adminNotes: String? = nil) async throws {
        guard let url = URL(string: "\(baseURL)/api/admin/requests") else { return }

        var payload: [String: Any] = ["id": id, "status": status]
        if let notes = adminNotes {
            payload["admin_notes"] = notes
        }

        do {
            var request = URLRequest(url: url)
            request.httpMethod = "PATCH"
            request.timeoutInterval = 4.0
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: payload)

            let (_, response) = try await URLSession.shared.data(for: request)
            if let http = response as? HTTPURLResponse, http.statusCode == 200 {
                // Succeeded
            }
        } catch {
            // In-memory update
        }

        if let idx = fallbackStore.firstIndex(where: { $0.id == id }) {
            fallbackStore[idx].status = status
            if let notes = adminNotes {
                fallbackStore[idx].admin_notes = notes
            }
        }
    }

    // MARK: - Delete Request
    public func deleteRequest(id: String) async throws {
        guard let url = URL(string: "\(baseURL)/api/admin/requests?id=\(id)") else { return }

        do {
            var request = URLRequest(url: url)
            request.httpMethod = "DELETE"
            request.timeoutInterval = 4.0
            let _ = try await URLSession.shared.data(for: request)
        } catch {}

        fallbackStore.removeAll(where: { $0.id == id })
    }

    private func filteredFallback(status: String?, inquiryType: String?) -> [EnterpriseRequest] {
        var list = fallbackStore
        if let s = status, s != "all" {
            list = list.filter { $0.status == s }
        }
        if let t = inquiryType, t != "all" {
            list = list.filter { $0.inquiry_type == t }
        }
        return list
    }
}
