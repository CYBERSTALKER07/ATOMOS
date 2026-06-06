//
//  HomeView.swift
//  driverappios
//

import SwiftUI
import CoreLocation
import MapKit

/// Default landing page — driver summary, today's stats, quick actions
struct HomeView: View {
    @Bindable var vm: FleetViewModel
    let onOpenMap: () -> Void

    @State private var appeared = false
    @State private var showNotificationInbox = false

    var body: some View {
        ScrollView {

            VStack(alignment: .leading, spacing: 20) {

                // MARK: - Greeting
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(vm.hasActiveRoute ? "MISSION ACTIVE" : greetingText)
                            .font(.system(size: 10, weight: .black, design: .monospaced)) // Tactical weight
                            .foregroundStyle(vm.hasActiveRoute ? LabTheme.live : LabTheme.fgTertiary)
                            .tracking(1.4) // Increased tracking

                        Text(vm.driverName)
                            .font(.system(size: 32, weight: .bold)) // Slightly larger
                            .foregroundStyle(LabTheme.fg)
                    }

                    Spacer()

                    Button { showNotificationInbox = true } label: {
                        Image(systemName: "bell")
                            .font(.system(size: 18, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .padding(.top, 8)
                }
                .padding(.horizontal, LabTheme.s4)
                .padding(.top, 60)

                // MARK: - Status Chip
                HStack(spacing: 10) {
                    statusChip(
                        icon: "truck.box.fill",
                        label: vm.licensePlate,
                        active: true
                    )
                    if vm.isReturning {
                        statusChip(
                            icon: "house.fill",
                            label: "Returning",
                            active: true,
                            tint: LabTheme.warning
                        )
                    } else {
                        statusChip(
                            icon: vm.hasActiveRoute ? "antenna.radiowaves.left.and.right" : "moon.zzz.fill",
                            label: vm.hasActiveRoute ? "On Route" : "Idle",
                            active: vm.hasActiveRoute
                        )
                    }
                }
                .staggeredAppear(index: 0)

                // MARK: - Vehicle Info Card
                if !vm.truckId.isEmpty && vm.truckId != "—" {
                    vehicleInfoCard
                        .staggeredAppear(index: 1)
                }

                // MARK: - Transit Control Card
                transitControlCard
                    .staggeredAppear(index: 2)

                // MARK: - Today Summary Card
                todaySummary
                    .staggeredAppear(index: 3)

                // MARK: - Open Map CTA
                mapButton
                    .staggeredAppear(index: 4)

                // MARK: - Quick Actions
                quickActions
                    .staggeredAppear(index: 5)

                // MARK: - Recent Activity
                recentActivity
                    .staggeredAppear(index: 6)
            }
            .padding(.horizontal, LabTheme.s16)
            .padding(.bottom, 20)
        }
        .scrollIndicators(.hidden)
        .background(LabTheme.bg)
        .sheet(isPresented: $showNotificationInbox) {
            DriverNotificationInboxView()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .task {
            await vm.loadMissions()
        }
    }

    // MARK: - Greeting

    private var greetingText: String {
        let hour = Calendar.current.component(.hour, from: Date())
        switch hour {
        case 5..<12:  return "GOOD MORNING"
        case 12..<17: return "GOOD AFTERNOON"
        case 17..<21: return "GOOD EVENING"
        default:      return "GOOD NIGHT"
        }
    }

    // MARK: - Status Chip

    private func statusChip(icon: String, label: String, active: Bool, tint: Color? = nil) -> some View {
        let chipColor = tint ?? (active ? LabTheme.fg : LabTheme.fgTertiary)
        return HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 11, weight: .black)) // Tactical bold
            Text(label.uppercased()) // Constant uppercase
                .font(.system(size: 11, weight: .bold, design: .monospaced))
        }
        .foregroundStyle(chipColor)
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background {
            RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous) // Using token
                .fill(LabTheme.card)
                .stroke(LabTheme.separator.opacity(0.12), lineWidth: 1) // Modern stroke
        }
    }

    // MARK: - Vehicle Info Card

    private var vehicleInfoCard: some View {
        HStack(spacing: 14) {
            Image(systemName: "truck.box.fill")
                .font(.system(size: 22))
                .foregroundStyle(LabTheme.fg)
                .frame(width: 44, height: 44)
                .background(LabTheme.separator)
                .clipShape(.rect(cornerRadius: 12))

            VStack(alignment: .leading, spacing: 4) {
                Text(vm.truckId)
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                HStack(spacing: 6) {
                    Text(vm.licensePlate)
                        .font(.system(size: 12, weight: .medium, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                    if vm.vehicleClass != "—" {
                        Text("•")
                            .foregroundStyle(LabTheme.fgTertiary)
                        Text("\(vm.vehicleClass) · \(Int(vm.maxVolumeVU)) VU")
                            .font(.system(size: 12, weight: .medium, design: .monospaced))
                            .foregroundStyle(LabTheme.fgTertiary)
                    }
                }
            }

            Spacer()

            Text("ASSIGNED")
                .font(.system(size: 9, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.success)
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .background(LabTheme.success.opacity(0.15))
                .clipShape(.capsule)
        }
        .padding(LabTheme.s16)
        .labCard()
    }

    // MARK: - Transit Control Card

    private var transitControlCard: some View {
        VStack(spacing: 14) {
            if vm.isReturning {
                // Returning to warehouse state
                HStack(spacing: 10) {
                    Circle()
                        .fill(LabTheme.warning)
                        .frame(width: 8, height: 8)
                        .modifier(PulseModifier())
                    Text("RETURNING TO WAREHOUSE")
                        .font(.system(size: 11, weight: .heavy, design: .monospaced))
                        .foregroundStyle(LabTheme.warning)
                    Spacer()
                }

                Text("All deliveries completed — head back to depot")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity, alignment: .leading)

                VStack(spacing: 8) {
                    Button {
                        Haptics.medium()
                        openWarehouseInMaps()
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: "arrow.triangle.turn.up.right.diamond.fill")
                                .font(.system(size: 14, weight: .semibold))
                            Text("NAVIGATE TO WAREHOUSE")
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
                            Text("ARRIVED AT WAREHOUSE")
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
                    Text("IN TRANSIT")
                        .font(.system(size: 11, weight: .heavy, design: .monospaced))
                        .foregroundStyle(LabTheme.live)
                    Spacer()
                    Text("\(vm.inTransitOrders.count) deliveries")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                }

                Text("Telemetry active — drive safely")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if !vm.loadedOrders.isEmpty {
                // Ready to depart
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("READY TO DEPART")
                            .font(.system(size: 11, weight: .heavy, design: .monospaced))
                            .foregroundStyle(LabTheme.fg)
                        Text("\(vm.loadedOrders.count) orders loaded")
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
                        Text("START TRANSIT")
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
                    Text("No orders loaded yet")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                    Spacer()
                }
            }
        }
        .padding(LabTheme.s20)
        .labCard()
    }

    // MARK: - Navigate to Warehouse

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

    // MARK: - Today Summary

    private var todaySummary: some View {
        VStack(spacing: 14) {
            HStack {
                Text("Today")
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                Spacer()
                Text(todayDate)
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
            }

            HStack(spacing: 0) {
                summaryTile(value: "\(vm.pendingMissions.count)", label: "Pending", icon: "clock")
                divider
                summaryTile(value: "\(vm.completedIds.count)", label: "Done", icon: "checkmark")
                divider
                summaryTile(value: totalRevenue, label: "Revenue", icon: "banknote")
            }
        }
        .padding(LabTheme.s20)
        .labCard()
    }

    private func summaryTile(value: String, label: String, icon: String) -> some View {
        VStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 12))
                .foregroundStyle(LabTheme.fgTertiary)
            Text(value)
                .font(.system(size: 18, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .lineLimit(1)
                .minimumScaleFactor(0.6)
            Text(label)
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
        }
        .frame(maxWidth: .infinity)
    }

    private var divider: some View {
        Rectangle()
            .fill(LabTheme.separator)
            .frame(width: 0.5, height: 36)
    }

    private var totalRevenue: String {
        let total = vm.completedMissions.reduce(0) { $0 + $1.amount }
        if total == 0 { return "—" }
        return total.formatted(.number.grouping(.automatic))
    }

    private var todayDate: String {
        Date().formatted(.dateTime.day().month(.abbreviated).year()).uppercased()
    }

    // MARK: - Map Button

    private var mapButton: some View {
        Button {
            Haptics.medium()
            onOpenMap()
        } label: {
            HStack(spacing: 14) {
                ZStack {
                    RoundedRectangle(cornerRadius: 14, style: .continuous)
                        .fill(LabTheme.fg)
                        .frame(width: 48, height: 48)

                    Image(systemName: "map.fill")
                        .font(.system(size: 18, weight: .semibold))
                        .foregroundStyle(LabTheme.buttonFg)
                }

                VStack(alignment: .leading, spacing: 3) {
                    Text("Open Map")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(LabTheme.fg)

                    Text("\(vm.pendingMissions.count) deliveries waiting")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Image(systemName: "arrow.right")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }

    // MARK: - Quick Actions

    private var quickActions: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Quick Actions")
                .font(.system(size: 14, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.horizontal, LabTheme.s4)

            HStack(spacing: 12) {
                actionTile(icon: "qrcode.viewfinder", label: "Scan QR")
                actionTile(icon: "shield.checkered", label: "Offline\nVerify")
                actionTile(icon: "arrow.triangle.2.circlepath", label: "Sync")
            }
        }
    }

    private func actionTile(icon: String, label: String) -> some View {
        VStack(spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 18, weight: .medium))
                .foregroundStyle(LabTheme.fg)

            Text(label)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(LabTheme.fgSecondary)
                .multilineTextAlignment(.center)
                .lineLimit(2)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 16)
        .labCard()
    }

    // MARK: - Recent Activity

    private var recentActivity: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Recent")
                .font(.system(size: 14, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.horizontal, LabTheme.s4)

            if vm.completedMissions.isEmpty {
                HStack {
                    Spacer()
                    VStack(spacing: 8) {
                        Image(systemName: "clock.arrow.circlepath")
                            .font(.system(size: 20))
                            .foregroundStyle(LabTheme.fgTertiary)
                        Text("No deliveries yet")
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .padding(.vertical, 24)
                    Spacer()
                }
                .labCard()
            } else {
                ForEach(vm.completedMissions.prefix(3)) { mission in
                    HStack(spacing: 12) {
                        ZStack {
                            Circle()
                                .fill(LabTheme.success.opacity(0.12))
                                .frame(width: 32, height: 32)
                            Image(systemName: "checkmark")
                                .font(.system(size: 11, weight: .bold))
                                .foregroundStyle(LabTheme.success)
                        }

                        VStack(alignment: .leading, spacing: 2) {
                            Text(mission.order_id)
                                .font(.system(size: 12, weight: .bold, design: .monospaced))
                                .foregroundStyle(LabTheme.fg)
                            Text(mission.amount.formattedAmount)
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(LabTheme.fgSecondary)
                        }

                        Spacer()

                        Text(mission.gateway)
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(LabTheme.fgTertiary)
                    }
                    .padding(LabTheme.s12)
                    .labCard()
                }
            }
        }
    }
}

// MARK: - Pulse Animation Modifier

private struct PulseModifier: ViewModifier {
    @State private var isPulsing = false

    func body(content: Content) -> some View {
        content
            .opacity(isPulsing ? 0.3 : 1.0)
            .animation(.easeInOut(duration: 1.0).repeatForever(autoreverses: true), value: isPulsing)
            .onAppear { isPulsing = true }
    }
}

#Preview {
    HomeView(vm: FleetViewModel(), onOpenMap: {})
}
