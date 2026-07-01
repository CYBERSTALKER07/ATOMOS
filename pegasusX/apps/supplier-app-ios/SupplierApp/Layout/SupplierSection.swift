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
    case earlyComplete = "Early complete"
    case orgFleet = "Org & fleet"
    case treasury = "Treasury"
    case retailerOverrides = "Retailer overrides"
    case chargebacks = "Chargebacks"
    case businessSetup = "Business setup"
    case inventoryImport = "Import inventory"
    case demandForecast = "Demand forecast"
    case planningBrain = "Planning sandbox"
    case planningSettings = "Planning settings"
    case knowledgeGraph = "Knowledge graph"
    case replenishmentPolicies = "Replenishment policies"
    case factories = "Factories"
    case warehouses = "Warehouses"
    case catalogDetail = "Catalog detail"

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
        case .earlyComplete: "checkmark.circle"
        case .orgFleet: "person.3"
        case .treasury: "building.columns"
        case .retailerOverrides: "tag.circle"
        case .chargebacks: "exclamationmark.bubble"
        case .businessSetup: "gearshape.2"
        case .inventoryImport: "square.and.arrow.down"
        case .demandForecast: "chart.xyaxis.line"
        case .planningBrain: "brain.head.profile"
        case .planningSettings: "calendar"
        case .knowledgeGraph: "point.3.connected.trianglepath.dotted"
        case .replenishmentPolicies: "doc.text"
        case .factories: "building.2"
        case .warehouses: "shippingbox.fill"
        case .catalogDetail: "square.grid.2x2.fill"
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
        [
            .manifests, .dispatchPreview, .activity,
            .fleetOrders, .orgFleet, .treasury, .ledger, .payments, .chargebacks,
            .reconciliation, .operations, .replenishmentPolicies,
        ]
    }

    static var intelligenceSections: [SupplierSection] {
        [.analytics, .aiRecommendations, .geoReport, .demandForecast, .planningBrain, .knowledgeGraph, .planningSettings]
    }

    static var networkSections: [SupplierSection] {
        [.topology, .factories, .warehouses, .deliveryZones, .supplyLanes]
    }

    static var accountSections: [SupplierSection] {
        [
            .catalog, .inventory, .inventoryImport, .promotions, .pricing, .retailerOverrides,
            .returns, .notifications, .earnings, .businessSetup, .profile,
        ]
    }
}
