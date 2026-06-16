import SwiftUI

enum SupplierSection: String, CaseIterable, Identifiable {
    case dashboard = "Dashboard"
    case orders = "Orders"
    case fleet = "Fleet"
    case exceptions = "Exceptions"
    case shopClosed = "Shop closed"
    case negotiations = "Negotiations"
    case manifests = "Manifests"
    case dispatchPreview = "Dispatch"
    case activity = "Activity"
    case fleetOrders = "Fleet orders"
    case ledger = "Ledger"
    case payments = "Payments"
    case operations = "Operations"
    case analytics = "Analytics"
    case aiRecommendations = "AI recommendations"
    case geoReport = "Geo report"
    case topology = "Topology"
    case deliveryZones = "Delivery zones"
    case supplyLanes = "Supply lanes"
    case catalog = "Catalog"
    case inventory = "Inventory"
    case promotions = "Promotions"
    case pricing = "Pricing"
    case returns = "Returns"
    case reconciliation = "Reconciliation"
    case notifications = "Notifications"
    case earnings = "Earnings"
    case profile = "Profile"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .dashboard: "square.grid.2x2"
        case .orders: "shippingbox"
        case .fleet: "truck.box"
        case .exceptions: "exclamationmark.triangle"
        case .shopClosed: "storefront"
        case .negotiations: "hand.raised"
        case .manifests: "doc.text"
        case .dispatchPreview: "paperplane"
        case .activity: "clock.arrow.circlepath"
        case .fleetOrders: "truck.box.fill"
        case .ledger: "banknote"
        case .payments: "creditcard"
        case .operations: "wrench.and.screwdriver"
        case .analytics: "chart.bar"
        case .aiRecommendations: "sparkles"
        case .geoReport: "map"
        case .topology: "building.2.crop.circle"
        case .deliveryZones: "mappin.and.ellipse"
        case .supplyLanes: "arrow.triangle.swap"
        case .catalog: "square.grid.2x2"
        case .inventory: "archivebox"
        case .promotions: "tag"
        case .pricing: "dollarsign.circle"
        case .returns: "arrow.uturn.backward"
        case .reconciliation: "scalemass"
        case .notifications: "bell"
        case .earnings: "chart.line.uptrend.xyaxis"
        case .profile: "building.2"
        }
    }

    /// Primary tabs on iPhone.
    static var compactTabs: [SupplierSection] {
        [.dashboard, .orders, .fleet]
    }

    /// iPad sidebar: primary + ops + intelligence + network + account.
    static var sidebarSections: [SupplierSection] {
        compactTabs + opsSections + intelligenceSections + networkSections + accountSections
    }

    static var opsSections: [SupplierSection] {
        // Quantity negotiation disabled ecosystem-wide.
        [.exceptions, .shopClosed, .manifests, .dispatchPreview, .activity, .fleetOrders, .ledger, .payments, .reconciliation, .operations]
    }

    static var intelligenceSections: [SupplierSection] {
        [.analytics, .aiRecommendations, .geoReport]
    }

    static var networkSections: [SupplierSection] {
        [.topology, .deliveryZones, .supplyLanes]
    }

    static var accountSections: [SupplierSection] {
        [.catalog, .inventory, .promotions, .pricing, .returns, .notifications, .earnings, .profile]
    }
}
