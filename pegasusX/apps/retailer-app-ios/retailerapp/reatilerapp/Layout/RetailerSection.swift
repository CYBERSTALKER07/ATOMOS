import SwiftUI

/// Canonical retailer navigation destinations — aligned with desktop routes, Android sidebar, and `ContentView` `SideMenuTab` / `AppTab`.
enum RetailerSection: String, CaseIterable, Identifiable {
    case dashboard = "Dashboard"
    case orders = "Orders"
    case tracking = "Tracking"
    case dock = "Dock"
    case catalog = "Catalog"
    case procurement = "Procurement"
    case insights = "Insights"
    case suppliers = "Suppliers"
    case autoOrder = "Auto Order"
    case futureDemand = "Future Demand"
    case notifications = "Notifications"
    case settings = "Settings"
    case cards = "Cards"
    case family = "Family"

    var id: String { rawValue }

    var desktopRoute: String {
        switch self {
        case .dashboard: "/dashboard"
        case .orders: "/orders"
        case .tracking: "/tracking"
        case .dock: "/dock"
        case .catalog: "/catalog"
        case .procurement: "/procurement"
        case .insights: "/insights"
        case .suppliers: "/catalog"
        case .autoOrder: "/settings"
        case .futureDemand: "/dashboard"
        case .notifications: "/notifications"
        case .settings: "/settings"
        case .cards: "/settings/cards"
        case .family: "/settings/family"
        }
    }

    var icon: String {
        switch self {
        case .dashboard: "house"
        case .orders: "shippingbox"
        case .tracking: "map"
        case .dock: "shippingbox.and.arrow.down"
        case .catalog: "square.grid.2x2"
        case .procurement: "chart.bar"
        case .insights: "chart.bar.xaxis"
        case .suppliers: "building.2"
        case .autoOrder: "arrow.2.squarepath"
        case .futureDemand: "waveform.path.ecg"
        case .notifications: "bell"
        case .settings: "gearshape"
        case .cards: "creditcard"
        case .family: "person.3"
        }
    }
}
