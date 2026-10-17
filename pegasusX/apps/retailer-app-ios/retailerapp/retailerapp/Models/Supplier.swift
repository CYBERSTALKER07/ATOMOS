import Foundation

// MARK: - Supplier

struct Supplier: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    let logoURL: String?
    let category: String?
    let orderCount: Int
    let productCount: Int
    let lastOrderDate: String?
    let phone: String?
    let email: String?
    let address: String?
    let primaryCategoryID: String?
    let operatingCategoryIDs: [String]
    let operatingCategoryNames: [String]
    /// Computed by the backend: false when ManualOffShift is true OR outside operating hours.
    let isActive: Bool
    let manualOffShift: Bool

    enum CodingKeys: String, CodingKey {
        case id, name, category, phone, email, address
        case logoURL = "logo_url"
        case logoURLCamel = "logoURL"
        case orderCount = "order_count"
        case orderCountCamel = "orderCount"
        case productCount = "product_count"
        case productCountCamel = "productCount"
        case lastOrderDate = "last_order_date"
        case lastOrderDateCamel = "lastOrderDate"
        case primaryCategoryID = "primary_category_id"
        case operatingCategoryIDs = "operating_category_ids"
        case operatingCategoryNames = "operating_category_names"
        case isActive = "is_active"
        case isActiveCamel = "isActive"
        case manualOffShift = "manual_off_shift"
        case manualOffShiftCamel = "manualOffShift"
    }

    init(
        id: String,
        name: String,
        logoURL: String?,
        category: String?,
        orderCount: Int,
        productCount: Int = 0,
        lastOrderDate: String?,
        phone: String?,
        email: String?,
        address: String?,
        primaryCategoryID: String? = nil,
        operatingCategoryIDs: [String] = [],
        operatingCategoryNames: [String] = [],
        isActive: Bool = true,
        manualOffShift: Bool = false
    ) {
        self.id = id
        self.name = name
        self.logoURL = logoURL
        self.category = category
        self.orderCount = orderCount
        self.productCount = productCount
        self.lastOrderDate = lastOrderDate
        self.phone = phone
        self.email = email
        self.address = address
        self.primaryCategoryID = primaryCategoryID
        self.operatingCategoryIDs = operatingCategoryIDs
        self.operatingCategoryNames = operatingCategoryNames
        self.isActive = isActive
        self.manualOffShift = manualOffShift
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        logoURL = try container.decodeIfPresent(String.self, forKey: .logoURL)
            ?? container.decodeIfPresent(String.self, forKey: .logoURLCamel)
        category = try container.decodeIfPresent(String.self, forKey: .category)
        orderCount = try container.decodeIfPresent(Int.self, forKey: .orderCount)
            ?? container.decodeIfPresent(Int.self, forKey: .orderCountCamel)
            ?? 0
        productCount = try container.decodeIfPresent(Int.self, forKey: .productCount)
            ?? container.decodeIfPresent(Int.self, forKey: .productCountCamel)
            ?? 0
        lastOrderDate = try container.decodeIfPresent(String.self, forKey: .lastOrderDate)
            ?? container.decodeIfPresent(String.self, forKey: .lastOrderDateCamel)
        phone = try container.decodeIfPresent(String.self, forKey: .phone)
        email = try container.decodeIfPresent(String.self, forKey: .email)
        address = try container.decodeIfPresent(String.self, forKey: .address)
        primaryCategoryID = try container.decodeIfPresent(String.self, forKey: .primaryCategoryID)
        operatingCategoryIDs = try container.decodeIfPresent([String].self, forKey: .operatingCategoryIDs) ?? []
        operatingCategoryNames = try container.decodeIfPresent([String].self, forKey: .operatingCategoryNames) ?? []
        isActive = try container.decodeIfPresent(Bool.self, forKey: .isActive)
            ?? container.decodeIfPresent(Bool.self, forKey: .isActiveCamel)
            ?? true
        manualOffShift = try container.decodeIfPresent(Bool.self, forKey: .manualOffShift)
            ?? container.decodeIfPresent(Bool.self, forKey: .manualOffShiftCamel)
            ?? false
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encodeIfPresent(logoURL, forKey: .logoURL)
        try container.encodeIfPresent(category, forKey: .category)
        try container.encode(orderCount, forKey: .orderCount)
        try container.encode(productCount, forKey: .productCount)
        try container.encodeIfPresent(lastOrderDate, forKey: .lastOrderDate)
        try container.encodeIfPresent(phone, forKey: .phone)
        try container.encodeIfPresent(email, forKey: .email)
        try container.encodeIfPresent(address, forKey: .address)
        try container.encodeIfPresent(primaryCategoryID, forKey: .primaryCategoryID)
        try container.encode(operatingCategoryIDs, forKey: .operatingCategoryIDs)
        try container.encode(operatingCategoryNames, forKey: .operatingCategoryNames)
        try container.encode(isActive, forKey: .isActive)
        try container.encode(manualOffShift, forKey: .manualOffShift)
    }

    var catalogSubtitle: String {
        if productCount > 0 {
            return "\(productCount) products"
        }
        return "\(orderCount) orders"
    }

    var displayCategory: String? {
        if let category, !category.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return category
        }
        let categories = operatingCategoryNames.filter { !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
        guard let first = categories.first else { return nil }
        switch categories.count {
        case 1:
            return first
        case 2:
            return categories.joined(separator: " • ")
        default:
            return "\(first) +\(categories.count - 1) more"
        }
    }

    var initials: String {
        let words = name.split(separator: " ")
        if words.count >= 2 {
            return String(words[0].prefix(1)) + String(words[1].prefix(1))
        }
        return String(name.prefix(2)).uppercased()
    }
}

extension Supplier {
    static let samples: [Supplier] = [
        Supplier(id: "sup-001", name: "Coca-Cola", logoURL: nil, category: "Beverages", orderCount: 14, lastOrderDate: "2026-03-15", phone: "+998901234567", email: "sales@coca-cola.uz", address: "Tashkent, Mirzo Ulugbek 42"),
        Supplier(id: "sup-002", name: "Nestlé", logoURL: nil, category: "Dairy", orderCount: 9, lastOrderDate: "2026-03-12", phone: "+998901234568", email: "orders@nestle.uz", address: "Tashkent, Shaykhantahur 15"),
        Supplier(id: "sup-003", name: "PepsiCo", logoURL: nil, category: "Beverages", orderCount: 7, lastOrderDate: "2026-03-10", phone: "+998901234569", email: "b2b@pepsico.uz", address: nil),
        Supplier(id: "sup-004", name: "Unilever", logoURL: nil, category: "Household", orderCount: 5, lastOrderDate: "2026-03-08", phone: "+998901234570", email: nil, address: nil),
        Supplier(id: "sup-005", name: "Local Farms Co.", logoURL: nil, category: "Produce", orderCount: 11, lastOrderDate: "2026-03-14", phone: "+998901234571", email: "hello@localfarms.uz", address: "Samarkand, Registan 7"),
        Supplier(id: "sup-006", name: "Baker's Best", logoURL: nil, category: "Bakery", orderCount: 3, lastOrderDate: "2026-03-06", phone: nil, email: nil, address: nil),
    ]
}
