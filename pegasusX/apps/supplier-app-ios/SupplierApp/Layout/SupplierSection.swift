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
    case operations = "Operations"
    case inventory = "Inventory"
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
        case .operations: "wrench.and.screwdriver"
        case .inventory: "archivebox"
        case .earnings: "chart.line.uptrend.xyaxis"
        case .profile: "building.2"
        }
    }

    /// Primary tabs on iPhone.
    static var compactTabs: [SupplierSection] {
        [.dashboard, .orders, .fleet]
    }

    /// iPad sidebar: primary + ops (account reachable via More hub on phone).
    static var sidebarSections: [SupplierSection] {
        compactTabs + opsSections + accountSections
    }

    static var opsSections: [SupplierSection] {
        [.exceptions, .shopClosed, .negotiations, .manifests, .dispatchPreview, .activity, .fleetOrders, .ledger, .operations]
    }

    static var accountSections: [SupplierSection] {
        [.inventory, .earnings, .profile]
    }
}
