import SwiftUI
import MapKit
import CoreLocation

struct LocationPickerView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var locationManager = LocationManager()

    var initialLatitude: Double
    var initialLongitude: Double
    var onConfirm: (Double, Double, String) -> Void

    @State private var cameraPosition: MapCameraPosition
    @State private var centerCoordinate: CLLocationCoordinate2D

    init(
        initialLatitude: Double = 0.0,
        initialLongitude: Double = 0.0,
        onConfirm: @escaping (Double, Double, String) -> Void
    ) {
        self.initialLatitude = initialLatitude
        self.initialLongitude = initialLongitude
        self.onConfirm = onConfirm

        let coord: CLLocationCoordinate2D
        if initialLatitude != 0 || initialLongitude != 0 {
            coord = CLLocationCoordinate2D(latitude: initialLatitude, longitude: initialLongitude)
        } else {
            // Default: Tashkent city center
            coord = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        }
        _centerCoordinate = State(initialValue: coord)
        _cameraPosition = State(initialValue: .region(MKCoordinateRegion(
            center: coord,
            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
        )))
    }

    var body: some View {
        ZStack {
            // Map
            Map(position: $cameraPosition) {
                // Center pin handled as overlay
            }
            .mapStyle(.standard(elevation: .realistic))
            .onMapCameraChange(frequency: .onEnd) { context in
                centerCoordinate = context.camera.centerCoordinate
            }
            .ignoresSafeArea()

            // Center pin
            VStack(spacing: 0) {
                Image(systemName: "mappin.circle.fill")
                    .font(.system(size: 36))
                    .foregroundStyle(.black)
                    .shadow(color: .black.opacity(0.3), radius: 4, y: 2)

                Image(systemName: "arrowtriangle.down.fill")
                    .font(.system(size: 10))
                    .foregroundStyle(.black)
                    .offset(y: -4)
            }
            .offset(y: -20) // pin tip at center

            // Top bar
            VStack {
                HStack {
                    Button {
                        dismiss()
                    } label: {
                        Image(systemName: "xmark")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                            .frame(width: 36, height: 36)
                            .background(.ultraThinMaterial)
                            .clipShape(.circle)
                    }

                    Spacer()

                    Text("Pick Store Location")
                        .font(.system(.headline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)

                    Spacer()

                    // Balance spacer
                    Color.clear.frame(width: 36, height: 36)
                }
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.top, 8)

                Spacer()
            }

            // My Location FAB
            VStack {
                Spacer()
                HStack {
                    Spacer()
                    Button {
                        requestAndMoveToUserLocation()
                    } label: {
                        Image(systemName: "location.fill")
                            .font(.system(size: 18, weight: .semibold))
                            .foregroundStyle(.black)
                            .frame(width: 48, height: 48)
                            .background(.white)
                            .clipShape(.circle)
                            .shadow(color: .black.opacity(0.15), radius: 8, y: 4)
                    }
                    .padding(.trailing, 16)
                }
                .padding(.bottom, 160)
            }

            // Bottom panel
            VStack {
                Spacer()
                VStack(spacing: AppTheme.spacingSM) {
                    Text(String(format: "%.5f, %.5f", centerCoordinate.latitude, centerCoordinate.longitude))
                        .font(.system(.body, design: .monospaced))
                        .foregroundStyle(AppTheme.textSecondary)

                    Text("Drag map to adjust pin position")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    Button {
                        let displayText = String(format: "%.5f, %.5f", centerCoordinate.latitude, centerCoordinate.longitude)
                        onConfirm(centerCoordinate.latitude, centerCoordinate.longitude, displayText)
                        dismiss()
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: "checkmark")
                                .font(.system(size: 14, weight: .semibold))
                            Text("Confirm Location")
                                .font(.system(.headline, design: .rounded))
                        }
                        .foregroundStyle(.white)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(AppTheme.accentGradient)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                    }
                    .padding(.top, 4)
                }
                .padding(AppTheme.spacingXL)
                .background(.ultraThinMaterial)
                .clipShape(.rect(cornerRadius: 24))
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.bottom, AppTheme.spacingMD)
            }
        }
    }

    private func requestAndMoveToUserLocation() {
        if locationManager.authorizationStatus == .notDetermined {
            locationManager.requestPermission()
            return
        }

        locationManager.requestLocation()

        // Animate to user location when available
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
            if let loc = locationManager.lastLocation {
                withAnimation(.easeInOut(duration: 0.6)) {
                    cameraPosition = .region(MKCoordinateRegion(
                        center: loc,
                        span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
                    ))
                }
            }
        }
    }
}
