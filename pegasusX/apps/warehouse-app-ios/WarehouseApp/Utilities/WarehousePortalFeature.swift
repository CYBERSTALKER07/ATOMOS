import Foundation

/// Portal-only surfaces — native apps hand off to warehouse-portal (port 3002).
enum WarehousePortalFeature: String, CaseIterable, Identifiable {
    case register
    case setup
    case profile
    case search

    var id: String { rawValue }

    var title: String {
        switch self {
        case .register: "Register warehouse"
        case .setup: "Warehouse setup"
        case .profile: "Profile"
        case .search: "Global search"
        }
    }

    var subtitle: String {
        switch self {
        case .register: "Create a new warehouse account"
        case .setup: "Location, billing, and configuration"
        case .profile: "Account and warehouse identity"
        case .search: "Jump to any portal page"
        }
    }

    var portalPath: String {
        switch self {
        case .register: "/auth/register"
        case .setup: "/setup/location"
        case .profile: "/profile"
        case .search: "/"
        }
    }

    var systemImage: String {
        switch self {
        case .register: "building.2"
        case .setup: "gearshape.2"
        case .profile: "person.crop.circle"
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
        case .search:
            "Global search (⌘K) is available on the warehouse web portal desktop shell."
        }
    }
}
