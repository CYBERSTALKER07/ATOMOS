import CoreLocation
import Foundation
import Observation

enum LoginMode: Equatable {
    case otp
    case passwordDev
}

@Observable
@MainActor
final class LoginViewModel {
    var mode: LoginMode = .otp
    var phone: String = "+998"
    var otpCode: String = ""
    var password: String = ""
    var otpSent: Bool = false
    var loading: Bool = false
    var error: String?

    private let configureAuth: @MainActor () -> Void
    private let resetAuthFlow: @MainActor () -> Void
    private let sendVerification: @MainActor (String) async throws -> Void
    private let verifyCode: @MainActor (String) async throws -> String
    private let loginWithIdToken: @MainActor (String) async throws -> AuthResponse
    private let loginWithPassword: @MainActor (String, String) async throws -> AuthResponse

    init(
        configureAuth: @escaping @MainActor () -> Void = { FirebaseAuthHelper.shared.configure() },
        resetAuthFlow: @escaping @MainActor () -> Void = { FirebaseAuthHelper.shared.resetFlow() },
        sendVerification: @escaping @MainActor (String) async throws -> Void = { try await FirebaseAuthHelper.shared.sendPhoneVerification(phone: $0) },
        verifyCode: @escaping @MainActor (String) async throws -> String = { try await FirebaseAuthHelper.shared.verifySmsCode($0) },
        loginWithIdToken: @escaping @MainActor (String) async throws -> AuthResponse = { try await FactoryService.login(idToken: $0) },
        loginWithPassword: @escaping @MainActor (String, String) async throws -> AuthResponse = { try await FactoryService.login(phone: $0, password: $1) }
    ) {
        self.configureAuth = configureAuth
        self.resetAuthFlow = resetAuthFlow
        self.sendVerification = sendVerification
        self.verifyCode = verifyCode
        self.loginWithIdToken = loginWithIdToken
        self.loginWithPassword = loginWithPassword
    }

    var canSendOtp: Bool {
        !loading && !phone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var canVerifyOtp: Bool {
        !loading && otpCode.count == 6
    }

    var canPasswordLogin: Bool {
        !loading
            && !phone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !password.isEmpty
    }

    func configure() {
        configureAuth()
    }

    func sanitizeOtp(_ raw: String) {
        let filtered = String(raw.filter(\.isNumber).prefix(6))
        if otpCode != filtered {
            otpCode = filtered
        }
    }

    func switchMode() {
        PegasusHaptics.selection()
        mode = mode == .otp ? .passwordDev : .otp
        error = nil
        otpSent = false
        otpCode = ""
        password = ""
        resetAuthFlow()
    }

    func resetForResend() {
        PegasusHaptics.selection()
        otpSent = false
        otpCode = ""
        error = nil
        resetAuthFlow()
    }

    func sendOtp() async {
        guard canSendOtp else { return }
        loading = true
        error = nil
        do {
            try await sendVerification(phone)
            otpSent = true
            PegasusHaptics.confirm()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        loading = false
    }

    func verifyOtp(onAuthenticated: (AuthResponse) -> Void) async {
        guard canVerifyOtp else { return }
        loading = true
        error = nil
        do {
            let idToken = try await verifyCode(otpCode)
            let auth = try await loginWithIdToken(idToken)
            PegasusHaptics.confirm()
            onAuthenticated(auth)
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        loading = false
    }

    func passwordLogin(onAuthenticated: (AuthResponse) -> Void) async {
        guard canPasswordLogin else { return }
        loading = true
        error = nil
        do {
            let auth = try await loginWithPassword(phone, password)
            PegasusHaptics.confirm()
            onAuthenticated(auth)
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        loading = false
    }
}

@Observable
@MainActor
final class LocationSetupViewModel {
    var factoryName: String = ""
    var location = AddressLocationValue()
    var loading: Bool = false
    var submitting: Bool = false
    var error: String?

    private let fetchLocation: @MainActor () async throws -> FactoryLocationResponse
    private let patchLocation: @MainActor (String, String?, Double, Double) async throws -> FactoryLocationResponse
    private let refreshAuth: @MainActor () async throws -> AuthResponse
    private let setupFactory: @MainActor (String, String, String?, Double, Double) async throws -> FactorySetupResponse
    private let resolveLocation: @MainActor (AddressLocationValue) async -> AddressLocationValue?

    init(
        fetchLocation: @escaping @MainActor () async throws -> FactoryLocationResponse = { try await FactoryService.factoryLocation() },
        patchLocation: @escaping @MainActor (String, String?, Double, Double) async throws -> FactoryLocationResponse = {
            try await FactoryService.patchFactoryLocation(address: $0, placeId: $1, lat: $2, lng: $3)
        },
        refreshAuth: @escaping @MainActor () async throws -> AuthResponse = { try await FactoryService.refresh() },
        setupFactory: @escaping @MainActor (String, String, String?, Double, Double) async throws -> FactorySetupResponse = {
            try await FactoryService.setup(factoryName: $0, address: $1, placeId: $2, lat: $3, lng: $4)
        },
        resolveLocation: @escaping @MainActor (AddressLocationValue) async -> AddressLocationValue? = {
            await GeocodeLocationSupport.resolveLocationValue($0)
        }
    ) {
        self.fetchLocation = fetchLocation
        self.patchLocation = patchLocation
        self.refreshAuth = refreshAuth
        self.setupFactory = setupFactory
        self.resolveLocation = resolveLocation
    }

    func loadExistingIfNeeded(hasAssignedFactory: Bool) async {
        guard hasAssignedFactory else { return }
        loading = true
        defer { loading = false }
        do {
            let resp = try await fetchLocation()
            factoryName = resp.name
            location = AddressLocationValue(
                address: resp.address,
                lat: resp.lat,
                lng: resp.lng,
                placeId: resp.placeId
            )
        } catch {
            self.error = error.localizedDescription
        }
    }

    func submit(
        hasAssignedFactory: Bool,
        currentRefreshToken: String?,
        onUpdateTokens: (String, String, String?) -> Void
    ) async {
        submitting = true
        error = nil
        defer { submitting = false }

        guard let resolved = await resolveLocation(location) else {
            error = "Select an address from the suggestions or share your location."
            PegasusHaptics.warning()
            return
        }
        location = resolved

        do {
            if hasAssignedFactory {
                _ = try await patchLocation(resolved.address, resolved.placeId, resolved.lat, resolved.lng)
                let auth = try await refreshAuth()
                onUpdateTokens(auth.token, auth.refreshToken, auth.factoryId)
                PegasusHaptics.confirm()
            } else {
                let trimmed = factoryName.trimmingCharacters(in: .whitespacesAndNewlines)
                guard trimmed.count >= 3 else {
                    error = "Factory name is required."
                    PegasusHaptics.warning()
                    return
                }
                let resp = try await setupFactory(trimmed, resolved.address, resolved.placeId, resolved.lat, resolved.lng)
                guard let token = resp.token, !token.isEmpty else {
                    error = "Setup failed."
                    PegasusHaptics.error()
                    return
                }
                onUpdateTokens(token, resp.refreshToken ?? currentRefreshToken ?? "", resp.factoryId)
                PegasusHaptics.confirm()
            }
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class LocationSettingsViewModel {
    var factoryName: String = ""
    var location = AddressLocationValue()
    var loading: Bool = true
    var saving: Bool = false
    var error: String?
    var saveMessage: String?

    private let fetchLocation: @MainActor () async throws -> FactoryLocationResponse
    private let patchLocation: @MainActor (String, String?, Double, Double) async throws -> FactoryLocationResponse
    private let resolveLocation: @MainActor (AddressLocationValue) async -> AddressLocationValue?

    init(
        fetchLocation: @escaping @MainActor () async throws -> FactoryLocationResponse = { try await FactoryService.factoryLocation() },
        patchLocation: @escaping @MainActor (String, String?, Double, Double) async throws -> FactoryLocationResponse = {
            try await FactoryService.patchFactoryLocation(address: $0, placeId: $1, lat: $2, lng: $3)
        },
        resolveLocation: @escaping @MainActor (AddressLocationValue) async -> AddressLocationValue? = {
            await GeocodeLocationSupport.resolveLocationValue($0)
        }
    ) {
        self.fetchLocation = fetchLocation
        self.patchLocation = patchLocation
        self.resolveLocation = resolveLocation
    }

    func load() async {
        loading = true
        error = nil
        do {
            let resp = try await fetchLocation()
            factoryName = resp.name
            location = AddressLocationValue(
                address: resp.address,
                lat: resp.lat,
                lng: resp.lng,
                placeId: resp.placeId
            )
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }

    func save() async {
        saving = true
        saveMessage = nil
        defer { saving = false }

        guard let resolved = await resolveLocation(location) else {
            saveMessage = "Select an address from the suggestions or share your location."
            PegasusHaptics.warning()
            return
        }
        location = resolved
        do {
            _ = try await patchLocation(resolved.address, resolved.placeId, resolved.lat, resolved.lng)
            saveMessage = "Location saved"
            PegasusHaptics.confirm()
            await load()
        } catch let saveError {
            saveMessage = saveError.localizedDescription
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class AddressLocationViewModel {
    var query: String = ""
    var suggestions: [GeocodePrediction] = []
    var error: String?
    var locating: Bool = false
    var resolving: Bool = false

    private var searchTask: Task<Void, Never>?

    func syncAddress(_ address: String) {
        if query != address {
            query = address
        }
    }

    func cancelSearch() {
        searchTask?.cancel()
        searchTask = nil
    }

    func scheduleSearch(_ text: String) {
        searchTask?.cancel()
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 3 else {
            suggestions = []
            return
        }
        searchTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: 280_000_000)
            guard !Task.isCancelled, let self else { return }
            await self.search(trimmed)
        }
    }

    private func search(_ text: String) async {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 3 else {
            suggestions = []
            return
        }
        let results = (try? await GeocodeService.autocomplete(trimmed)) ?? []
        guard !Task.isCancelled else { return }
        suggestions = results
    }

    func pick(placeId: String, fallback: String, current: AddressLocationValue) async -> AddressLocationValue {
        PegasusHaptics.selection()
        let id = placeId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else {
            return await resolveText(fallback, current: current) ?? current
        }
        do {
            let loc = try await GeocodeService.resolvePlace(id)
            return apply(loc, fallback: fallback)
        } catch {
            return await resolveText(fallback, current: current) ?? current
        }
    }

    func resolveOnBlur(current: AddressLocationValue) async -> AddressLocationValue? {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        if trimmed == current.address.trimmingCharacters(in: .whitespacesAndNewlines),
           GeocodeLocationSupport.hasValidCoordinates(lat: current.lat, lng: current.lng) {
            return nil
        }
        return await resolveText(trimmed, current: current)
    }

    func useMyLocation() async -> AddressLocationValue? {
        PegasusHaptics.selection()
        locating = true
        defer { locating = false }
        let manager = CLLocationManager()
        manager.requestWhenInUseAuthorization()
        var coord = manager.location?.coordinate
        if coord == nil || (coord?.latitude == 0 && coord?.longitude == 0) {
            coord = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        }
        guard let target = coord else {
            error = "Current location unavailable."
            return nil
        }
        do {
            let loc = try await GeocodeService.reverse(lat: target.latitude, lng: target.longitude)
            return apply(loc, fallback: query.isEmpty ? "Amir Temur Ave, Tashkent" : query)
        } catch {
            return apply(
                ResolvedLocation(address: "Amir Temur Ave, Tashkent, Uzbekistan", lat: target.latitude, lng: target.longitude, placeId: nil),
                fallback: "Amir Temur Ave, Tashkent, Uzbekistan"
            )
        }
    }

    private func resolveText(_ text: String, current: AddressLocationValue) async -> AddressLocationValue? {
        resolving = true
        defer { resolving = false }
        guard let resolved = await GeocodeLocationSupport.resolveLocationValue(
            AddressLocationValue(address: text, lat: current.lat, lng: current.lng, placeId: current.placeId)
        ) else {
            return nil
        }
        query = resolved.address
        suggestions = []
        error = nil
        return resolved
    }

    private func apply(_ loc: ResolvedLocation, fallback: String) -> AddressLocationValue {
        let next = AddressLocationValue(
            address: loc.address.isEmpty ? fallback : loc.address,
            lat: loc.lat,
            lng: loc.lng,
            placeId: loc.placeId
        )
        query = next.address
        suggestions = []
        error = nil
        return next
    }
}
