import Foundation

public struct EnterpriseRequest: Identifiable, Codable, Sendable, Hashable {
    public let id: String
    public let name: String
    public let email: String
    public let company: String?
    public let phone: String?
    public let role: String?
    public let fleet_size: String?
    public let inquiry_type: String
    public let subject: String?
    public let message: String?
    public var status: String
    public let priority: String
    public var admin_notes: String?
    public let source_page: String?
    public let created_at: String
    public let updated_at: String?

    public init(
        id: String = UUID().uuidString,
        name: String,
        email: String,
        company: String? = nil,
        phone: String? = nil,
        role: String? = "Supplier Operations",
        fleet_size: String? = nil,
        inquiry_type: String = "demo",
        subject: String? = nil,
        message: String? = nil,
        status: String = "new",
        priority: String = "normal",
        admin_notes: String? = nil,
        source_page: String? = "desktop_macos",
        created_at: String = ISO8601DateFormatter().string(from: Date()),
        updated_at: String? = nil
    ) {
        self.id = id
        self.name = name
        self.email = email
        self.company = company
        self.phone = phone
        self.role = role
        self.fleet_size = fleet_size
        self.inquiry_type = inquiry_type
        self.subject = subject
        self.message = message
        self.status = status
        self.priority = priority
        self.admin_notes = admin_notes
        self.source_page = source_page
        self.created_at = created_at
        self.updated_at = updated_at
    }

    public var displayDate: String {
        let isoFormatter = ISO8601DateFormatter()
        isoFormatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = isoFormatter.date(from: created_at) {
            let outputFormatter = DateFormatter()
            outputFormatter.dateStyle = .medium
            outputFormatter.timeStyle = .short
            return outputFormatter.string(from: date)
        }

        // Fallback standard ISO
        let fallback = ISO8601DateFormatter()
        if let date = fallback.date(from: created_at) {
            let outputFormatter = DateFormatter()
            outputFormatter.dateStyle = .medium
            outputFormatter.timeStyle = .short
            return outputFormatter.string(from: date)
        }

        return created_at
    }
}

public struct EnterpriseAPIResponse<T: Codable>: Codable {
    public let success: Bool
    public let data: T?
    public let count: Int?
    public let error: String?
    public let message: String?
}
