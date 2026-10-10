import Foundation

enum RegisterStep: Int, CaseIterable {
    case identity
    case verification
    case profile

    var title: String {
        switch self {
        case .identity: "Phone verification"
        case .verification: "Confirm code"
        case .profile: "Basic profile"
        }
    }
}

struct RegisterCountry: Identifiable {
    let id: String
    let name: String
    let dialCode: String
    let currency: String
}

@Observable
@MainActor
final class OnboardingViewModel {
    static let countries: [RegisterCountry] = [
        .init(id: "UZ", name: "Uzbekistan", dialCode: "+998", currency: "UZS"),
        .init(id: "KZ", name: "Kazakhstan", dialCode: "+7", currency: "KZT"),
        .init(id: "KG", name: "Kyrgyzstan", dialCode: "+996", currency: "KGS"),
        .init(id: "AE", name: "United Arab Emirates", dialCode: "+971", currency: "AED"),
        .init(id: "TR", name: "Türkiye", dialCode: "+90", currency: "TRY"),
        .init(id: "US", name: "United States", dialCode: "+1", currency: "USD"),
    ]

    var step: RegisterStep = .identity
    var countryCode = "UZ"
    var phoneLocal = ""
    var otpCode = ""
    var legalName = ""
    var contactName = ""
    var email = ""
    var password = ""

    var taxId = ""
    var registrationNumber = ""
    var headquartersAddress = ""
    var city = ""
    var postalCode = ""

    var loading = false
    var error: String?

    var dialCode: String {
        Self.countries.first { $0.id == countryCode }?.dialCode ?? "+998"
    }

    var fullPhone: String { "\(dialCode)\(phoneLocal)" }

    func advanceStep() -> Bool {
        switch step {
        case .identity:
            guard phoneLocal.count >= 6 else {
                error = "Enter a valid phone number."
                return false
            }
            step = .verification
        case .verification:
            guard otpCode.count == 6 else {
                error = "Enter the 6-digit code."
                return false
            }
            step = .profile
        case .profile:
            return true
        }
        error = nil
        return true
    }

    func retreatStep() {
        guard let prior = RegisterStep(rawValue: step.rawValue - 1) else { return }
        step = prior
        error = nil
    }

    func register(tokenStore: TokenStore) async -> Bool {
        guard legalName.count >= 2, contactName.count >= 2, email.contains("@") else {
            error = "Complete all profile fields."
            return false
        }
        loading = true
        error = nil
        defer { loading = false }
        do {
            let body: [String: Any] = [
                "account": [
                    "legalName": legalName,
                    "contactName": contactName,
                    "email": email,
                    "country": countryCode,
                    "phone": fullPhone,
                ],
                "id_token": otpCode,
            ]
            let response = try await SupplierService.register(body: body)
            tokenStore.storeRegister(response)
            Task { await PushNotificationManager.shared.uploadStoredTokenIfPossible() }
            return true
        } catch {
            self.error = error.localizedDescription
            return false
        }
    }

    func setupBusiness(tokenStore: TokenStore) async -> Bool {
        guard taxId.count >= 5, headquartersAddress.count >= 5, city.count >= 2 else {
            error = "Tax ID, address, and city are required."
            return false
        }
        loading = true
        error = nil
        defer { loading = false }
        do {
            let response = try await SupplierService.setupBusiness(
                BusinessSetupRequest(
                    taxId: taxId,
                    registrationNumber: registrationNumber,
                    headquartersAddress: headquartersAddress,
                    city: city,
                    postalCode: postalCode
                )
            )
            tokenStore.markRegistered(response.isRegistered)
            return true
        } catch {
            self.error = error.localizedDescription
            return false
        }
    }
}
