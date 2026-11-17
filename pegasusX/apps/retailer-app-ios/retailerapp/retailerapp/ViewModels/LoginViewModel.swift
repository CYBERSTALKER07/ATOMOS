import Foundation
import SwiftUI
import Observation
import CoreLocation

@Observable
@MainActor
final class LoginViewModel {
    var phone = "+998"
    var password = ""
    var isLoginMode = true

    // Registration fields
    var storeName = ""
    var ownerName = ""
    var addressText = ""
    var taxId = ""
    var receivingWindowOpen = ""
    var receivingWindowClose = ""
    var accessType = ""           // STREET_PARKING | ALLEYWAY | LOADING_DOCK
    var storageCeilingHeightText = ""

    // Location & presentation state
    let locationManager = LocationManager()
    var latitude: Double = 0.0
    var longitude: Double = 0.0
    var locationLabel = ""
    var showLocationPicker = false
    var locating = false
    var logoScale: Double = 0.5
    var formOpacity: Double = 0

    var phoneInput: String {
        get { phone }
        set {
            if newValue.hasPrefix("+998") || newValue == "+99" || newValue == "+9" || newValue == "+" {
                phone = newValue
            }
        }
    }

    var isFormValid: Bool {
        guard phone.count >= 13, !password.isEmpty else { return false }
        if !isLoginMode {
            return !storeName.isEmpty && !ownerName.isEmpty && !addressText.isEmpty
        }
        return true
    }

    func setLocation(lat: Double, lng: Double, label: String) {
        latitude = lat
        longitude = lng
        locationLabel = label
        if addressText.isEmpty {
            addressText = label
        }
    }

    func toggleAccessType(_ value: String) {
        Haptics.selection()
        accessType = accessType == value ? "" : value
    }

    func shareCurrentLocation() async {
        guard !locating else { return }
        Haptics.light()
        locating = true
        locationManager.requestLocation()
        for _ in 0..<20 {
            if Task.isCancelled {
                locating = false
                return
            }
            if let loc = locationManager.lastLocation {
                let resolved = await GeocodeService.reverse(lat: loc.latitude, lng: loc.longitude)
                let label = resolved ?? String(format: "%.5f, %.5f", loc.latitude, loc.longitude)
                setLocation(lat: loc.latitude, lng: loc.longitude, label: label)
                locating = false
                Haptics.success()
                return
            }
            try? await Task.sleep(nanoseconds: 100_000_000)
        }
        let fallback = LocationManager.defaultLocation
        let resolved = await GeocodeService.reverse(lat: fallback.latitude, lng: fallback.longitude)
        let label = resolved ?? "Amir Temur Square, Tashkent"
        setLocation(lat: fallback.latitude, lng: fallback.longitude, label: label)
        locating = false
        Haptics.selection()
    }

    func submit(auth: AuthManager) async {
        if isLoginMode {
            await auth.login(phone: phone, password: password)
        } else {
            await auth.register(
                phone: phone,
                password: password,
                storeName: storeName,
                ownerName: ownerName,
                addressText: addressText,
                latitude: latitude,
                longitude: longitude,
                taxId: taxId.isEmpty ? nil : taxId,
                receivingWindowOpen: receivingWindowOpen.isEmpty ? nil : receivingWindowOpen,
                receivingWindowClose: receivingWindowClose.isEmpty ? nil : receivingWindowClose,
                accessType: accessType.isEmpty ? nil : accessType,
                storageCeilingHeightCM: Double(storageCeilingHeightText)
            )
        }
        if auth.errorMessage == nil {
            Haptics.success()
        } else {
            Haptics.error()
        }
    }
}
