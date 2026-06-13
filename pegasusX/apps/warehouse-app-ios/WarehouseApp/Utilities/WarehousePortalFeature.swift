import Foundation

/// Portal-only surfaces — native apps hand off to warehouse-portal (port 3002).
enum WarehousePortalFeature: String, CaseIterable, Identifiable {
    case register
    case setup
    case profile
    case notifications
    case search

    var id: String { rawValue }

    var title: String {
        switch self {
        case .register: "Register warehouse"
        case .setup: "Warehouse setup"
        case .profile: "Profile"
        case .notifications: "Notifications"
        case .search: "Global search"
        }
    }

    var subtitle: String {
        switch self {
        case .register: "Create a new warehouse account"
        case .setup: "Location, billing, and configuration"
        case .profile: "Account and warehouse identity"
        case .notifications: "Alerts and operational inbox"
        case .search: "Jump to any portal page"
        }
    }

    var portalPath: String {
        switch self {
        case .register: "/auth/register"
        case .setup: "/setup/location"
        case .profile: "/profile"
        case .notifications, .search: "/"
        }
    }

    var systemImage: String {
        switch self {
        case .register: "building.2"
        case .setup: "gearshape.2"
        case .profile: "person.crop.circle"
        case .notifications: "bell"
        case .search: "magnifyingglass"
        }
    }

    var handoffMessage: String {
        switch self {
        case .register:
            "New warehouse registration is completed on the warehouse web portal."
        case .setup:
            "Warehouse setup and onboarding run on the web portal after registration."
        case .profile:
            "Profile and account settings are managed on the warehouse web portal."
        case .notifications:
            "The notification panel lives in the warehouse portal top bar. Open the portal to review alerts."
        case .search:
            "Global search (⌘K) is available on the warehouse web portal desktop shell."
        }
    }
}
