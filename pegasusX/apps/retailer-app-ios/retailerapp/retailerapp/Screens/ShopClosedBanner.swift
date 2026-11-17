import PhotosUI
import SwiftUI

struct ShopClosedBanner: View {
    let event: ShopClosedAlertEvent
    let onRespond: (String) -> Void
    @State private var socket = RetailerWebSocket.shared
    @State private var vm = ShopClosedBannerViewModel()

    var body: some View {
        @Bindable var vm = vm
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(.yellow)
                Text(L10n.format("mobile_retailer.ui.driver_drivername_reported_your_shop_is_closed", "\(event.driverName)"))
                    .font(.headline)
            }
            .accessibilityElement(children: .combine)

            Text("mobile_retailer.ui.please_confirm_your_status_so_we_can_manage_your_order")
                .font(.subheadline)
                .foregroundStyle(.secondary)

            if let errorText = vm.errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            if vm.bypassPending {
                Text("mobile_retailer.ui.doorway_drop_off_proof_is_required_for_authorize_bypass")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                PhotosPicker(selection: $vm.photoItem, matching: .images) {
                    Label(
                        vm.uploading ? "Uploading…" : (vm.bypassPhotoURL != nil ? "Replace photo" : "Take or choose photo"),
                        systemImage: "camera"
                    )
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.secondary.opacity(0.15))
                    .clipShape(.rect(cornerRadius: 8))
                    .pegasusMinimumHitTarget()
                }
                .disabled(vm.isSubmitting || vm.uploading)
                .onChange(of: vm.photoItem) { _, item in
                    guard let item else { return }
                    Task { await vm.uploadBypassPhoto(item, orderId: event.orderId) }
                }
                if vm.bypassPhotoURL != nil {
                    Button {
                        vm.respond(
                            with: "AUTHORIZE_BYPASS",
                            orderId: event.orderId,
                            photoURL: vm.bypassPhotoURL,
                            onRespond: onRespond
                        )
                    } label: {
                        Text("mobile_retailer.ui.confirm_bypass")
                            .pegasusMinimumHitTarget()
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(vm.isSubmitting || vm.uploading)
                }
                Button {
                    vm.cancelBypass()
                } label: {
                    Text("mobile_retailer.ui.cancel_bypass")
                        .pegasusMinimumHitTarget()
                }
                .disabled(vm.isSubmitting)
            }

            VStack(spacing: 8) {
                ForEach(event.options, id: \.self) { option in
                    let optionLabel = vm.labelForOption(option)
                    Button {
                        vm.selectOption(option, orderId: event.orderId, onRespond: onRespond)
                    } label: {
                        Text(optionLabel)
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color.blue)
                            .foregroundStyle(.white)
                            .clipShape(.rect(cornerRadius: 8))
                            .pegasusMinimumHitTarget()
                    }
                    .disabled(vm.isSubmitting || vm.uploading)
                    .accessibilityLabel(optionLabel)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .clipShape(.rect(cornerRadius: 12))
        .shadow(radius: 5)
        .padding()
        .onChange(of: socket.reconnectEpoch) { _, _ in
            vm.handleReconnect()
        }
    }
}
