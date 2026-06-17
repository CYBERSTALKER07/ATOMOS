import Foundation

@Observable
final class OnboardingViewModel {
    enum SetupStep {
        case tax
        case address
    }

    var step: SetupStep = .tax
    var taxId = ""
    var billingAddress = ""
    var shippingAddress = ""
    var city = ""
    var postalCode = ""
    var isSubmitting = false
    var errorMessage: String?

    private let api = APIClient.shared

    func validateTax() -> String? {
        taxId.trimmingCharacters(in: .whitespacesAndNewlines).count < 5
            ? "Tax ID is required (min 5 characters)"
            : nil
    }

    func validateAddress() -> String? {
        if billingAddress.trimmingCharacters(in: .whitespacesAndNewlines).count < 5 {
            return "Billing address is required"
        }
        if shippingAddress.trimmingCharacters(in: .whitespacesAndNewlines).count < 5 {
            return "Shipping address is required"
        }
        if city.trimmingCharacters(in: .whitespacesAndNewlines).count < 2 {
            return "City is required"
        }
        return nil
    }

    func advanceFromTax() -> Bool {
        if let error = validateTax() {
            errorMessage = error
            return false
        }
        errorMessage = nil
        step = .address
        return true
    }

    func submit() async -> Bool {
        if let error = validateAddress() {
            errorMessage = error
            return false
        }
        isSubmitting = true
        errorMessage = nil
        defer { isSubmitting = false }

        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        let payload: [String: AnyEncodable] = [
            "tax_id": AnyEncodable(taxId.trimmingCharacters(in: .whitespacesAndNewlines)),
            "billing_address": AnyEncodable(billingAddress.trimmingCharacters(in: .whitespacesAndNewlines)),
            "shipping_address": AnyEncodable(shippingAddress.trimmingCharacters(in: .whitespacesAndNewlines)),
            "city": AnyEncodable(city.trimmingCharacters(in: .whitespacesAndNewlines)),
            "postal_code": AnyEncodable(postalCode.trimmingCharacters(in: .whitespacesAndNewlines)),
        ]

        do {
            try await api.setupRetailer(
                payload: payload,
                idempotencyKey: "retailer-setup:\(retailerId)"
            )
            AuthManager.shared.markConfigured()
            return true
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Setup access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry setup.",
                fallback: "Setup failed. Please try again."
            )
            return false
        }
    }
}
