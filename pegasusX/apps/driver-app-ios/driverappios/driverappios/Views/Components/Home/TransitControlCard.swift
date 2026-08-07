import SwiftUI
import CoreLocation
import MapKit

struct TransitControlCard: View {
    var vm: FleetViewModel

    var body: some View {
        VStack(spacing: 14) {
            if vm.isReturning {
                // Returning to warehouse state
                HStack(spacing: 10) {
                    Circle()
                        .fill(LabTheme.warning)
                        .frame(width: 8, height: 8)
                        .modifier(PulseModifier())
                    Text("mobile_driver.ui.returning_to_warehouse")
                        .font(.system(size: 11, weight: .heavy, design: .monospaced))
                        .foregroundStyle(LabTheme.warning)
                    Spacer()
                }

                Text("mobile_driver.ui.all_deliveries_completed_head_back_to_depot")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity, alignment: .leading)

                if vm.returnGoodsTotalUnits > 0 {
                    Text(L10n.format("mobile_driver.ui.returngoodstotalunits_item_s_to_return_on_truck", "\(vm.returnGoodsTotalUnits)"))
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(LabTheme.warning)
                        .frame(maxWidth: .infinity, alignment: .leading)
                    ForEach(vm.returnGoodsLines.prefix(5)) { line in
                        Text(L10n.format("mobile_driver.ui.productname_quantity", "\(line.productName)", "\(line.quantity)"))
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }

                if vm.showCashReconSheet || vm.deliveryEdgeMessage?.localizedCaseInsensitiveContains("cash reconciliation") == true {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("mobile_driver.ui.cash_reconciliation")
                            .font(.system(size: 12, weight: .semibold))
                        TextField("mobile_driver.ui.declared_cash_minor", text: Bindable(vm).declaredCashText)
                            .textFieldStyle(.roundedBorder)
                        Button("mobile_driver.ui.submit_reconciliation") {
                            Task { await vm.submitCashReconciliation() }
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }

                VStack(spacing: 8) {
                    Button {
                        Haptics.medium()
                        openWarehouseInMaps()
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: "arrow.triangle.turn.up.right.diamond.fill")
                                .font(.system(size: 14, weight: .semibold))
                            Text("mobile_driver.ui.navigate_to_warehouse")
                                .font(.system(size: 13, weight: .heavy, design: .monospaced))
                        }
                        .foregroundStyle(LabTheme.bg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(LabTheme.warning)
                        .clipShape(.rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .buttonStyle(.pressable)

                    Button {
                        Task { await vm.returnComplete() }
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: "house.fill")
                                .font(.system(size: 14, weight: .semibold))
                            Text("mobile_driver.ui.arrived_at_warehouse")
                                .font(.system(size: 13, weight: .heavy, design: .monospaced))
                        }
                        .foregroundStyle(LabTheme.fg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(LabTheme.fg.opacity(0.08))
                        .clipShape(.rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .buttonStyle(.pressable)
                }
            } else if vm.isTransitActive {
                // Active transit state
                HStack(spacing: 10) {
                    Circle()
                        .fill(LabTheme.live)
                        .frame(width: 8, height: 8)
                        .modifier(PulseModifier())
                    Text("mobile_driver.ui.in_transit")
                        .font(.system(size: 11, weight: .heavy, design: .monospaced))
                        .foregroundStyle(LabTheme.live)
                    Spacer()
                    Text(L10n.format("mobile_driver.ui.count_deliveries", "\(vm.inTransitOrders.count)"))
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                }

                Text("mobile_driver.ui.telemetry_active_drive_safely")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if !vm.loadedOrders.isEmpty {
                // Ready to depart
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("mobile_driver.ui.ready_to_depart")
                            .font(.system(size: 11, weight: .heavy, design: .monospaced))
                            .foregroundStyle(LabTheme.fg)
                        Text(L10n.format("mobile_driver.ui.count_orders_loaded", "\(vm.loadedOrders.count)"))
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(LabTheme.fgTertiary)
                    }
                    Spacer()
                }

                Button {
                    Task { await vm.departRoute() }
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "truck.box.fill")
                            .font(.system(size: 14, weight: .semibold))
                        Text("mobile_driver.ui.start_transit")
                            .font(.system(size: 13, weight: .heavy, design: .monospaced))
                    }
                    .foregroundStyle(LabTheme.bg)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(LabTheme.fg)
                    .clipShape(.rect(cornerRadius: LabTheme.buttonRadius))
                }
                .buttonStyle(.pressable)
            } else {
                // No orders loaded
                HStack(spacing: 10) {
                    Image(systemName: "tray")
                        .font(.system(size: 14))
                        .foregroundStyle(LabTheme.fgTertiary)
                    Text("mobile_driver.ui.no_orders_loaded_yet")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                    Spacer()
                }
            }
        }
        .padding(LabTheme.s20)
        .labCard()
    }

    private func openWarehouseInMaps() {
        let lat = TokenStore.shared.warehouseLat != 0 ? TokenStore.shared.warehouseLat : 41.2995
        let lng = TokenStore.shared.warehouseLng != 0 ? TokenStore.shared.warehouseLng : 69.2401
        let depotCoord = CLLocationCoordinate2D(latitude: lat, longitude: lng)
        let placemark = MKPlacemark(coordinate: depotCoord)
        let mapItem = MKMapItem(placemark: placemark)
        mapItem.name = TokenStore.shared.warehouseName ?? "Warehouse"
        mapItem.openInMaps(launchOptions: [
            MKLaunchOptionsDirectionsModeKey: MKLaunchOptionsDirectionsModeDriving
        ])
    }
}

// Ensure PulseModifier is available, either move here or keep in HomeView if it's reused. It's only used here and in HomeView for the PulseStrip... Wait, PulseStrip is a separate view? Let's check HomeView line 597: `PulseModifier`. Let's just create a shared one or duplicate for simplicity. Actually, I'll extract PulseModifier to its own file or keep it in TransitControlCard if not used in HomeView. Ah, HomeView line 54 uses PulseStrip, which is another component. So PulseModifier might only be used in TransitControlCard!
