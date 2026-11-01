import Foundation

public enum RolePerspective: String, CaseIterable, Identifiable, Codable, Sendable {
    case overall = "Overall Ecosystem"
    case supplier = "Supplier Perspective"
    case warehouse = "Warehouse Perspective"
    case retailer = "Retailer Perspective"
    case fleet = "Fleet & Carrier"

    public var id: String { rawValue }

    public var systemIcon: String {
        switch self {
        case .overall: return "globe.europe.africa.fill"
        case .supplier: return "shippingbox.fill"
        case .warehouse: return "building.2.fill"
        case .retailer: return "storefront.fill"
        case .fleet: return "truck.box.fill"
        }
    }
}

public struct TenderWaterfallSlice: Identifiable, Codable, Sendable {
    public var id: String { label }
    public let label: String
    public let amountTiyin: Int64
    public let percentage: Double
    public let isDeduction: Bool
    public let detail: String

    public var amountUzsFormatted: String {
        let uzs = Double(amountTiyin) / 100.0
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.maximumFractionDigits = 0
        let formatted = formatter.string(from: NSNumber(value: uzs)) ?? "\(Int64(uzs))"
        return "\(formatted) UZS"
    }

    public init(label: String, amountTiyin: Int64, percentage: Double, isDeduction: Bool, detail: String) {
        self.label = label
        self.amountTiyin = amountTiyin
        self.percentage = percentage
        self.isDeduction = isDeduction
        self.detail = detail
    }
}

public struct AttributedCostItem: Identifiable, Codable, Sendable {
    public var id: String { "\(role)-\(entityId)-\(resourceName)" }
    public let role: String
    public let entityId: String
    public let resourceName: String
    public let monthlyCostUsd: Double
    public let usageMetric: String
    public let trendPercent: Double

    public init(
        role: String,
        entityId: String,
        resourceName: String,
        monthlyCostUsd: Double,
        usageMetric: String,
        trendPercent: Double
    ) {
        self.role = role
        self.entityId = entityId
        self.resourceName = resourceName
        self.monthlyCostUsd = monthlyCostUsd
        self.usageMetric = usageMetric
        self.trendPercent = trendPercent
    }
}

public struct BillingInvoice: Identifiable, Codable, Sendable {
    public var id: String { invoiceNumber }
    public let invoiceNumber: String
    public let entityId: String
    public let role: String
    public let period: String
    public let totalAmountTiyin: Int64
    public let status: InvoiceStatus
    public let generatedAt: String

    public enum InvoiceStatus: String, Codable, Sendable {
        case paid = "PAID"
        case pending = "PENDING"
        case processing = "PROCESSING"
    }

    public var amountUzsFormatted: String {
        let uzs = Double(totalAmountTiyin) / 100.0
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.maximumFractionDigits = 0
        return "\(formatter.string(from: NSNumber(value: uzs)) ?? "\(Int64(uzs))") UZS"
    }

    public init(
        invoiceNumber: String,
        entityId: String,
        role: String,
        period: String,
        totalAmountTiyin: Int64,
        status: InvoiceStatus,
        generatedAt: String
    ) {
        self.invoiceNumber = invoiceNumber
        self.entityId = entityId
        self.role = role
        self.period = period
        self.totalAmountTiyin = totalAmountTiyin
        self.status = status
        self.generatedAt = generatedAt
    }
}
