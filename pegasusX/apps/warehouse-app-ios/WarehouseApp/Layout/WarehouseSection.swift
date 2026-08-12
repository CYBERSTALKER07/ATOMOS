import SwiftUI

enum WarehouseSection: String, CaseIterable, Identifiable {
    case dashboard = "Dashboard"
    case orders = "Orders"
    case drivers = "Drivers"
    case vehicles = "Trucks"
    case inventory = "Inventory"
    case dispatch = "Dispatch"
    case analytics = "Analytics"
    case treasury = "Treasury"
    case staff = "Staff"
    case manifests = "Manifests"
    case dispatchSettings = "Dispatch settings"
    case fleetLiveMap = "Live fleet"
    case transferActions = "Transfer actions"
    case products = "Products"
    case supplyRequests = "Supply requests"
    case preorders = "Pre-orders"
    case stockCommitments = "Stock commitments"
    case tomorrowBoard = "Tomorrow board"
    case replenishment = "Replenishment"
    case demandForecast = "Demand forecast"
    case retailers = "Retailers"
    case returns = "Returns"
    case coldChain = "Cold chain"
    case laborCapacity = "Labor capacity"
    case exceptions = "Exceptions"
    case claims = "Claims"
    case rescues = "Rescues"
    case paymentConfig = "Payment config"
    case opsSettings = "Ops settings"
    case returnPolicy = "Returns & reverse SLA"
    case notifications = "Notifications"
    case portalSetup = "Warehouse setup"
    case portalProfile = "Profile"
    case portalSearch = "Global search"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .dashboard: "square.grid.2x2"
        case .orders: "cart"
        case .drivers: "person.badge.key"
        case .vehicles: "truck.box"
        case .inventory: "archivebox"
        case .dispatch: "paperplane"
        case .analytics: "chart.bar.xaxis"
        case .treasury: "banknote"
        case .staff: "person.2"
        case .manifests: "doc.text"
        case .dispatchSettings: "slider.horizontal.3"
        case .fleetLiveMap: "map"
        case .transferActions: "arrow.left.arrow.right"
        case .products: "square.grid.2x2"
        case .supplyRequests: "arrow.triangle.2.circlepath"
        case .preorders: "calendar"
        case .stockCommitments: "chart.pie.fill"
        case .tomorrowBoard: "calendar.badge.clock"
        case .replenishment: "shippingbox"
        case .demandForecast: "chart.line.uptrend.xyaxis"
        case .retailers: "person.crop.rectangle"
        case .returns: "arrow.uturn.backward"
        case .coldChain: "thermometer"
        case .laborCapacity: "person.3"
        case .exceptions: "exclamationmark.triangle"
        case .claims: "doc.text"
        case .rescues: "wrench.and.screwdriver"
        case .paymentConfig: "creditcard"
        case .opsSettings: "gearshape"
        case .returnPolicy: "arrow.uturn.backward.circle"
        case .notifications: "bell"
        case .portalSetup: "gearshape.2"
        case .portalProfile: "person.crop.circle"
        case .portalSearch: "magnifyingglass"
        }
    }

    /// Primary iPhone tabs (compact shell).
    static var compactTabs: [WarehouseSection] {
        [.dashboard, .orders, .dispatch]
    }

    static var primarySections: [WarehouseSection] {
        compactTabs + [.drivers, .vehicles, .inventory, .analytics, .treasury, .staff]
    }

    static var fulfillmentSections: [WarehouseSection] {
        [.manifests, .dispatchSettings, .fleetLiveMap, .transferActions]
    }

    static var inventorySections: [WarehouseSection] {
        [.products, .supplyRequests, .preorders, .stockCommitments, .tomorrowBoard, .replenishment, .demandForecast, .opsSettings, .returnPolicy]
    }

    static var operationsSections: [WarehouseSection] {
        [.retailers, .returns, .coldChain, .laborCapacity, .exceptions, .claims, .rescues, .paymentConfig, .notifications]
    }

    static var portalSections: [WarehouseSection] {
        [.portalSetup, .portalProfile, .portalSearch]
    }

    /// iPad sidebar: mirrors warehouse-portal nav groups.
    static var sidebarSections: [WarehouseSection] {
        primarySections + fulfillmentSections + inventorySections + operationsSections + portalSections
    }

    var portalFeature: WarehousePortalFeature? {
        switch self {
        case .portalSetup: .setup
        case .portalProfile: .profile
        case .portalSearch: .search
        default: nil
        }
    }
}

enum WarehouseCompactTab: Hashable {
    case dashboard
    case orders
    case dispatch
    case more
}
