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
    case coverage = "Coverage and supply"
    case opsSettings = "Ops settings"
    case returnPolicy = "Returns & reverse SLA"
    case notifications = "Notifications"
    case controlTower = "Control tower"
    case portalSetup = "Warehouse setup"
    case portalProfile = "Profile"
    case portalSearch = "Global search"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .dashboard: "antenna.radiowaves.left.and.right"
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
        case .coldChain: "thermometer.snowflake"
        case .laborCapacity: "person.3"
        case .exceptions: "exclamationmark.triangle"
        case .claims: "doc.text"
        case .rescues: "wrench.and.screwdriver"
        case .paymentConfig: "creditcard"
        case .coverage: "mappin.and.ellipse"
        case .opsSettings: "gearshape"
        case .returnPolicy: "arrow.uturn.backward.circle"
        case .notifications: "bell"
        case .controlTower: "shield.lefthalf.filled"
        case .portalSetup: "gearshape.2"
        case .portalProfile: "person.crop.circle"
        case .portalSearch: "magnifyingglass"
        }
    }

    /// Primary iPhone tabs: Command · Inbound · Floor · Dispatch · More.
    static var compactTabs: [WarehouseSection] {
        [.dashboard, .manifests, .inventory, .dispatch]
    }

    static var primarySections: [WarehouseSection] {
        [.dashboard, .dispatch, .inventory, .demandForecast]
    }

    static var fulfillmentSections: [WarehouseSection] {
        [.manifests, .fleetLiveMap, .dispatchSettings, .transferActions]
    }

    static var inventorySections: [WarehouseSection] {
        [.products, .supplyRequests, .preorders, .stockCommitments, .tomorrowBoard, .replenishment, .coverage, .returnPolicy]
    }

    static var operationsSections: [WarehouseSection] {
        [.orders, .drivers, .vehicles, .coldChain, .controlTower, .exceptions, .rescues, .claims, .retailers, .returns, .laborCapacity, .analytics, .treasury, .staff, .paymentConfig, .opsSettings, .notifications]
    }

    static var portalSections: [WarehouseSection] {
        [.portalSetup, .portalProfile, .portalSearch]
    }

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
    case inbound
    case floor
    case dispatch
    case plan
    case more
}
