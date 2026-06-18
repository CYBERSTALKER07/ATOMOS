import SwiftUI

enum FactorySection: String, CaseIterable, Identifiable {
    case dashboard = "Dashboard"
    case loadingBay = "Loading Bay"
    case transfers = "Transfers"
    case fleet = "Fleet"
    case staff = "Staff"
    case location = "Location"
    case supplyRequests = "Supply requests"
    case payloadOverride = "Override"
    case manifests = "Manifests"
    case manifestExceptions = "Exceptions"
    case insights = "Insights"
    case analytics = "Analytics"
    case notifications = "Notifications"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .dashboard: "square.grid.2x2"
        case .loadingBay: "shippingbox"
        case .transfers: "arrow.left.arrow.right"
        case .fleet: "truck.box"
        case .staff: "person.2"
        case .location: "mappin.and.ellipse"
        case .supplyRequests: "checklist"
        case .payloadOverride: "arrow.left.arrow.right.square"
        case .manifests: "doc.text"
        case .manifestExceptions: "exclamationmark.triangle"
        case .insights: "chart.bar.xaxis"
        case .analytics: "chart.xyaxis.line"
        case .notifications: "bell"
        }
    }

    static var compactTabs: [FactorySection] { [.dashboard, .loadingBay, .transfers] }

    static var primarySections: [FactorySection] {
        [.dashboard, .loadingBay, .transfers, .fleet, .staff, .location]
    }

    static var operationsSections: [FactorySection] {
        [.supplyRequests, .payloadOverride, .manifests, .manifestExceptions]
    }

    static var intelligenceSections: [FactorySection] {
        [.insights, .analytics, .notifications]
    }
}

enum FactoryCompactTab: Hashable {
    case dashboard
    case loadingBay
    case transfers
    case more
}
