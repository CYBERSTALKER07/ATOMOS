import CoreLocation

// MARK: - Location Manager
// Uses CLLocationManager delegate for real GPS coordinates with reliable fallback for simulator and offline environments.

@Observable
final class LocationManager: NSObject, CLLocationManagerDelegate {
    static let defaultLocation = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)

    private let manager = CLLocationManager()

    var lastLocation: CLLocationCoordinate2D?
    var authorizationStatus: CLAuthorizationStatus
    var isLocating: Bool = false
    var locationError: String?

    override init() {
        authorizationStatus = manager.authorizationStatus
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyBest
    }

    func requestPermission() {
        manager.requestWhenInUseAuthorization()
    }

    func requestLocation() {
        isLocating = true
        locationError = nil
        authorizationStatus = manager.authorizationStatus

        if authorizationStatus == .notDetermined {
            manager.requestWhenInUseAuthorization()
            // On simulator, permission dialog might not resolve immediately or location might be empty;
            // don't leave isLocating hanging.
        } else if authorizationStatus == .denied || authorizationStatus == .restricted {
            isLocating = false
            if lastLocation == nil {
                lastLocation = Self.defaultLocation
            }
        } else {
            manager.requestLocation()
        }
    }

    // MARK: - CLLocationManagerDelegate

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        isLocating = false
        guard let latest = locations.last(where: {
            $0.horizontalAccuracy >= 0 && $0.horizontalAccuracy <= 100 && abs($0.timestamp.timeIntervalSinceNow) <= 30
        }) ?? locations.last(where: { $0.horizontalAccuracy >= 0 }) else {
            if lastLocation == nil {
                lastLocation = Self.defaultLocation
            }
            return
        }
        lastLocation = latest.coordinate
    }

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        authorizationStatus = manager.authorizationStatus
        if authorizationStatus == .authorizedWhenInUse || authorizationStatus == .authorizedAlways {
            manager.requestLocation()
        } else if authorizationStatus == .denied || authorizationStatus == .restricted {
            isLocating = false
            if lastLocation == nil {
                lastLocation = Self.defaultLocation
            }
        }
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        print("LocationManager: location request failed — \(error.localizedDescription)")
        isLocating = false
        locationError = error.localizedDescription
        // In simulator without custom location set or if hardware GPS fails, fall back to default location
        if lastLocation == nil {
            lastLocation = Self.defaultLocation
        }
    }
}
