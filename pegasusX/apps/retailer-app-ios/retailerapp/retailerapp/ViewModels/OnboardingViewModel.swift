import Foundation

@Observable
@MainActor
final class OnboardingViewModel {
    enum SetupStep {
        case tax
        case address
    }

    var step: SetupStep = .tax
    var taxId = ""
    var isInformalShop = false
    var innChecking = false
    var innLookup: RetailerINNLookupResponse?
    var innError: String?
    var joinRequestConflict: RetailerJoinRequestConflict?
    var joinRequestSent = false
    var billingAddress = ""
    var shippingAddress = ""
    var city = ""
    var postalCode = ""
    var isSubmitting = false
    var errorMessage: String?

    private let lookupINNHandler: (_ inn: String) async throws -> RetailerINNLookupResponse
    private let setupRetailerHandler: (_ payload: [String: AnyEncodable], _ idempotencyKey: String) async throws -> Void
    private let currentRetailerIdProvider: () -> String
    private let onConfiguredHandler: () -> Void

    init(
        lookupINN: @escaping (_ inn: String) async throws -> RetailerINNLookupResponse = { try await APIClient.shared.lookupINN(inn: $0) },
        setupRetailer: @escaping (_ payload: [String: AnyEncodable], _ idempotencyKey: String) async throws -> Void = { payload, key in
            try await APIClient.shared.setupRetailer(payload: payload, idempotencyKey: key)
        },
        currentRetailerId: @escaping () -> String = { AuthManager.shared.currentUser?.id ?? "" },
        onConfigured: @escaping () -> Void = { AuthManager.shared.markConfigured() }
    ) {
        self.lookupINNHandler = lookupINN
        self.setupRetailerHandler = setupRetailer
        self.currentRetailerIdProvider = currentRetailerId
        self.onConfiguredHandler = onConfigured
    }

    func validateTax() -> String? {
        if isInformalShop {
            return nil
        }
        guard RetailerW3ContractHelper.normalizedINNOrNil(taxId) != nil else {
            return String(localized: "supplier_portal.onboarding_inn.invalid_inn")
        }
        if let lookup = innLookup {
            if lookup.isUnavailable {
                return String(localized: "retailer_desktop.tax_identity.registry_unavailable")
            }
            if lookup.isInvalidOrMissing {
                return String(localized: "supplier_portal.onboarding_inn.invalid_inn")
            }
        }
        return nil
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

    func verifyINN() async -> Bool {
        innError = nil
        errorMessage = nil
        joinRequestConflict = nil
        joinRequestSent = false

        guard let cleanINN = RetailerW3ContractHelper.normalizedINNOrNil(taxId) else {
            innError = String(localized: "supplier_portal.onboarding_inn.invalid_inn")
            return false
        }

        innChecking = true
        defer { innChecking = false }

        do {
            let response = try await lookupINNHandler(cleanINN)
            innLookup = response
            if response.isUnavailable {
                innError = String(localized: "retailer_desktop.tax_identity.registry_unavailable")
                return false
            }
            if response.isInvalidOrMissing {
                innError = String(localized: "supplier_portal.onboarding_inn.invalid_inn")
                return false
            }
            return response.isVerifiedFound
        } catch {
            innLookup = RetailerINNLookupResponse(
                status: "UNAVAILABLE",
                normalizedInn: cleanINN,
                entityType: nil,
                legalName: nil,
                address: nil,
                registryState: nil,
                cached: nil,
                existingTenantId: nil,
                existingTenantType: nil,
                existingLegalName: nil,
                existingJoinEligible: nil
            )
            innError = String(localized: "retailer_desktop.tax_identity.registry_unavailable")
            return false
        }
    }

    func advanceFromTax() -> Bool {
        if let error = validateTax() {
            errorMessage = error
            return false
        }
        if !isInformalShop && innLookup?.isVerifiedFound != true {
            errorMessage = String(localized: "retailer_desktop.tax_identity.registry_unavailable")
            return false
        }
        errorMessage = nil
        step = .address
        return true
    }

    func advanceFromTaxAsync() async -> Bool {
        if isInformalShop {
            errorMessage = nil
            innError = nil
            step = .address
            return true
        }
        guard let cleanINN = RetailerW3ContractHelper.normalizedINNOrNil(taxId) else {
            let msg = String(localized: "supplier_portal.onboarding_inn.invalid_inn")
            innError = msg
            errorMessage = msg
            return false
        }
        if innLookup?.normalizedInn != cleanINN || innLookup?.isVerifiedFound != true {
            let verified = await verifyINN()
            if !verified {
                errorMessage = innError ?? String(localized: "retailer_desktop.tax_identity.registry_unavailable")
                return false
            }
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
        if !isInformalShop {
            if let taxErr = validateTax() {
                errorMessage = taxErr
                return false
            }
            let verifiedNow: Bool
            if let cleanINN = RetailerW3ContractHelper.normalizedINNOrNil(taxId),
               innLookup?.normalizedInn == cleanINN,
               innLookup?.isVerifiedFound == true {
                verifiedNow = true
            } else {
                verifiedNow = await verifyINN()
            }
            if !verifiedNow {
                errorMessage = innError ?? String(localized: "retailer_desktop.tax_identity.registry_unavailable")
                return false
            }
        }

        isSubmitting = true
        errorMessage = nil
        joinRequestConflict = nil
        joinRequestSent = false
        defer { isSubmitting = false }

        let retailerId = currentRetailerIdProvider()
        let cleanTaxId = isInformalShop ? "" : (RetailerW3ContractHelper.normalizedINNOrNil(taxId) ?? "")
        let payload: [String: AnyEncodable] = [
            "tax_id": AnyEncodable(cleanTaxId),
            "informal_shop": AnyEncodable(isInformalShop),
            "legal_name": AnyEncodable(innLookup?.legalName ?? ""),
            "billing_address": AnyEncodable(billingAddress.trimmingCharacters(in: .whitespacesAndNewlines)),
            "shipping_address": AnyEncodable(shippingAddress.trimmingCharacters(in: .whitespacesAndNewlines)),
            "city": AnyEncodable(city.trimmingCharacters(in: .whitespacesAndNewlines)),
            "postal_code": AnyEncodable(postalCode.trimmingCharacters(in: .whitespacesAndNewlines)),
        ]

        do {
            try await setupRetailerHandler(
                payload,
                RetailerIdempotency.setup(retailerId: retailerId)
            )
            onConfiguredHandler()
            return true
        } catch {
            if let conflict = RetailerW3ContractHelper.parseJoinRequestConflict(from: error) {
                joinRequestConflict = conflict
                joinRequestSent = true
                errorMessage = nil
                return false
            }
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
