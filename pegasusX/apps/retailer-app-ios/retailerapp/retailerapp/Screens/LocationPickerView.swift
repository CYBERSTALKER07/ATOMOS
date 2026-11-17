import SwiftUI
import MapKit
import CoreLocation

struct LocationPickerView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm: LocationPickerViewModel

    var initialLatitude: Double
    var initialLongitude: Double
    var onConfirm: (Double, Double, String) -> Void

    init(
        initialLatitude: Double = 0.0,
        initialLongitude: Double = 0.0,
        onConfirm: @escaping (Double, Double, String) -> Void
    ) {
        self.initialLatitude = initialLatitude
        self.initialLongitude = initialLongitude
        self.onConfirm = onConfirm
        _vm = State(initialValue: LocationPickerViewModel(
            initialLatitude: initialLatitude,
            initialLongitude: initialLongitude
        ))
    }

    var body: some View {
        @Bindable var vm = vm
        ZStack {
            Map(position: $vm.cameraPosition) {}
                .mapStyle(.standard(elevation: .realistic))
                .onMapCameraChange(frequency: .onEnd) { context in
                    vm.onCameraEnded(context.camera.centerCoordinate)
                }
                .ignoresSafeArea()

            VStack(spacing: 0) {
                Image(systemName: "mappin.circle.fill")
                    .font(.largeTitle)
                    .foregroundStyle(.black)
                    .shadow(color: .black.opacity(0.3), radius: 4, y: 2)

                Image(systemName: "arrowtriangle.down.fill")
                    .font(.caption2)
                    .foregroundStyle(.black)
                    .offset(y: -4)
            }
            .offset(y: -20)
            .allowsHitTesting(false)
            .accessibilityHidden(true)

            VStack {
                HStack {
                    Button {
                        dismiss()
                    } label: {
                        Image(systemName: "xmark")
                            .font(.body.weight(.semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                            .frame(width: 44, height: 44)
                            .background(.ultraThinMaterial)
                            .clipShape(.circle)
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Close location picker")

                    Spacer()

                    Text("mobile_retailer.ui.pick_store_location")
                        .font(.system(.headline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)

                    Spacer()

                    Color.clear.frame(width: 44, height: 44)
                }
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.top, 8)

                Spacer()
            }

            VStack {
                Spacer()
                HStack {
                    Spacer()
                    Button {
                        vm.requestAndMoveToUserLocation()
                    } label: {
                        Image(systemName: "location.fill")
                            .font(.headline.weight(.semibold))
                            .foregroundStyle(.black)
                            .frame(width: 48, height: 48)
                            .background(.white)
                            .clipShape(.circle)
                            .shadow(color: .black.opacity(0.15), radius: 8, y: 4)
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Move map to current location")
                    .padding(.trailing, 16)
                }
                .padding(.bottom, 160)
            }

            VStack {
                Spacer()
                VStack(spacing: AppTheme.spacingSM) {
                    Text(vm.statusDisplayText)
                        .font(.system(.body, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                        .multilineTextAlignment(.center)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Selected location: \(vm.statusDisplayText)")

                    Button {
                        Task {
                            let confirmed = await vm.confirmSelection()
                            onConfirm(confirmed.lat, confirmed.lng, confirmed.label)
                            dismiss()
                        }
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: "checkmark")
                                .font(.subheadline.weight(.semibold))
                            Text("mobile_retailer.ui.confirm_location")
                                .font(.system(.headline, design: .rounded))
                        }
                        .foregroundStyle(.white)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(AppTheme.accentGradient)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Confirm store location")
                    .padding(.top, 4)
                }
                .padding(AppTheme.spacingXL)
                .background(.ultraThinMaterial)
                .clipShape(.rect(cornerRadius: 24))
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.bottom, AppTheme.spacingMD)
            }
        }
        .task { await vm.resolveAddress() }
        .onChange(of: vm.locationManager.lastLocation?.latitude) { _, _ in
            vm.handleLocationChange()
        }
        .onDisappear {
            vm.cancelTasks()
        }
    }
}
