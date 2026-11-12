import Foundation
import SwiftUI
import Observation
import MapKit
import CoreLocation

@Observable
@MainActor
final class LocationPickerViewModel {
    let locationManager = LocationManager()
    var cameraPosition: MapCameraPosition
    var centerCoordinate: CLLocationCoordinate2D
    var addressLabel: String?
    var resolving = false
    var awaitingUserLocation = false

    private var resolveTask: Task<Void, Never>?
    private var locateTask: Task<Void, Never>?

    init(initialLatitude: Double = 0.0, initialLongitude: Double = 0.0) {
        let coord: CLLocationCoordinate2D
        if initialLatitude != 0 || initialLongitude != 0 {
            coord = CLLocationCoordinate2D(latitude: initialLatitude, longitude: initialLongitude)
        } else {
            let center = packMapCoordinate()
            coord = CLLocationCoordinate2D(latitude: center.lat, longitude: center.lng)
        }
        self.centerCoordinate = coord
        self.cameraPosition = .region(MKCoordinateRegion(
            center: coord,
            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
        ))
    }

    var statusDisplayText: String {
        if resolving {
            return "Resolving address…"
        }
        if let addressLabel, !addressLabel.isEmpty {
            return addressLabel
        }
        return "Drag map to adjust pin position"
    }

    func onCameraEnded(_ newCenter: CLLocationCoordinate2D) {
        centerCoordinate = newCenter
        scheduleResolveAddress()
    }

    func scheduleResolveAddress() {
        resolveTask?.cancel()
        resolveTask = Task {
            try? await Task.sleep(nanoseconds: 180_000_000)
            guard !Task.isCancelled else { return }
            await resolveAddress()
        }
    }

    func resolveAddress() async {
        resolving = true
        defer { resolving = false }
        let coord = centerCoordinate
        let resolved = await GeocodeService.reverse(lat: coord.latitude, lng: coord.longitude)
        guard !Task.isCancelled else { return }
        addressLabel = resolved
    }

    func handleLocationChange() {
        guard awaitingUserLocation, let loc = locationManager.lastLocation else { return }
        awaitingUserLocation = false
        withAnimation(.easeInOut(duration: 0.6)) {
            cameraPosition = .region(MKCoordinateRegion(
                center: loc,
                span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
            ))
        }
    }

    func requestAndMoveToUserLocation() {
        Haptics.light()
        awaitingUserLocation = true
        locationManager.requestLocation()
        locateTask?.cancel()
        locateTask = Task {
            for _ in 0..<20 {
                if Task.isCancelled { return }
                if let loc = locationManager.lastLocation {
                    awaitingUserLocation = false
                    withAnimation(.easeInOut(duration: 0.6)) {
                        cameraPosition = .region(MKCoordinateRegion(
                            center: loc,
                            span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
                        ))
                    }
                    centerCoordinate = loc
                    Haptics.selection()
                    await resolveAddress()
                    return
                }
                try? await Task.sleep(nanoseconds: 100_000_000)
            }
            guard !Task.isCancelled else { return }
            let fallback = LocationManager.defaultLocation
            awaitingUserLocation = false
            withAnimation(.easeInOut(duration: 0.6)) {
                cameraPosition = .region(MKCoordinateRegion(
                    center: fallback,
                    span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
                ))
            }
            centerCoordinate = fallback
            await resolveAddress()
        }
    }

    func confirmSelection() async -> (lat: Double, lng: Double, label: String) {
        let label: String
        if let addr = addressLabel, !addr.isEmpty {
            label = addr
        } else {
            label = await GeocodeService.reverse(
                lat: centerCoordinate.latitude,
                lng: centerCoordinate.longitude
            ) ?? "Pinned location"
        }
        Haptics.success()
        return (centerCoordinate.latitude, centerCoordinate.longitude, label)
    }

    func cancelTasks() {
        resolveTask?.cancel()
        locateTask?.cancel()
    }
}
