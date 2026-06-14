import Foundation

/// Portal-only supplier surfaces — native apps hand off to supplier-portal.
enum SupplierPortalFeature: String, CaseIterable, Identifiable {
    case register
    case businessSetup
    case chargebacks
    case paymentBypass

    var id: String { rawValue }

    var title: String {
        switch self {
        case .register: "Register supplier"
        case .businessSetup: "Business setup"
        case .chargebacks: "Chargebacks"
        case .paymentBypass: "Payment bypass"
        }
    }

    var portalPath: String {
        switch self {
        case .register: "/auth/register"
        case .businessSetup: "/setup/business"
        case .chargebacks: "/payments"
        case .paymentBypass: "/operations"
        }
    }

    var handoffMessage: String {
        switch self {
        case .register:
            "Supplier registration runs on the supplier web portal."
        case .businessSetup:
            "Business profile setup is completed on the supplier web portal."
        case .chargebacks:
            "Chargeback review and treasury actions are managed on the supplier portal."
        case .paymentBypass:
            "Payment bypass is available in Operations on native and web."
        }
    }
}
