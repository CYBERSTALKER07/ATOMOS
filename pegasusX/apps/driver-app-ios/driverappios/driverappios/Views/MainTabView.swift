//
//  MainTabView.swift
//  driverappios
//

import SwiftUI
import SwiftData
import Network

enum AppTab: CaseIterable {
    case home, map, rides, profile

    var icon: String {
        switch self {
        case .home:    "house.fill"
        case .map:     "map.fill"
        case .rides:   "list.bullet"
        case .profile: "person.fill"
        }
    }

    var label: String {
        switch self {
        case .home:    "Home"
        case .map:     "Map"
        case .rides:   "Rides"
        case .profile: "Profile"
        }
    }
}

struct MainTabView: View {
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var selectedTab: AppTab = .home
    @State private var vm = FleetViewModel()
    @State private var pathMonitor: NWPathMonitor?
    @State private var wasOffline = false

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                ipadCockpitLayout
            } else {
                phoneLayout
            }
        }
        .onChange(of: scenePhase) { _, newPhase in
            if newPhase == .active {
                Task { await vm.loadMissions(silent: true) }
            }
        }
        .onAppear {
            guard pathMonitor == nil else { return }
            let monitor = NWPathMonitor()
            let queue = DispatchQueue(label: "com.pegasus.driver.main-tab.network")
            monitor.pathUpdateHandler = { path in
                DispatchQueue.main.async {
                    if path.status == .satisfied {
                        if wasOffline {
                            Task {
                                await FleetServiceLive.shared.flushOfflineQueue()
                                await vm.loadMissions(silent: true)
                            }
                        }
                        wasOffline = false
                    } else {
                        wasOffline = true
                    }
                }
            }
            monitor.start(queue: queue)
            pathMonitor = monitor
        }
        .onDisappear {
            pathMonitor?.cancel()
            pathMonitor = nil
        }
        .alert("Rescue Proposed", isPresented: $vm.showRescueProposalAlert) {
            Button("mobile_driver.ui.accept_rescue", role: .none) {
                Task { await vm.acceptRescue() }
            }
            Button("mobile_driver.ui.reject", role: .cancel) {
                Task { await vm.rejectRescue() }
            }
        } message: {
            Text("mobile_driver.ui.a_nearby_truck_broke_down_do_you_accept_the_rescue")
        }
    }

    // MARK: - iPhone & Compact Split Layout
    private var phoneLayout: some View {
        Group {
            if selectedTab == .map {
                FleetMapView(vm: vm, goBack: {
                    withAnimation(Anim.snappy) { selectedTab = .home }
                })
            } else {
                VStack(spacing: 0) {
                    TabView(selection: $selectedTab) {
                        Tab("Home", systemImage: "house.fill", value: .home) {
                            HomeView(vm: vm, onOpenMap: {
                                withAnimation(Anim.snappy) { selectedTab = .map }
                            }, onHandoffNavigate: { dest in
                                switch dest {
                                case .home: selectedTab = .home
                                case .fleetMap: selectedTab = .map
                                case .manifestList, .manifestDetail, .orderDetail: selectedTab = .rides
                                case .unresolved: break
                                }
                            })
                        }

                        Tab("Rides", systemImage: "list.bullet", value: .rides) {
                            RidesListView(vm: vm)
                        }

                        Tab("Profile", systemImage: "person.fill", value: .profile) {
                            ProfileView(vm: vm)
                        }
                    }
                    .tabViewStyle(.tabBarOnly)
                    .tint(LabTheme.fg)
                }
                .safeAreaInset(edge: .bottom) {
                    if vm.hasActiveRoute, let mission = vm.activeMission {
                        ActiveRideBar(
                            mission: mission,
                            driverLocation: vm.location,
                            onTap: {
                                withAnimation(Anim.snappy) { selectedTab = .map }
                            }
                        )
                        .labReadableWidth()
                        .padding(.horizontal, LabTheme.s16)
                        .padding(.bottom, LabTheme.s8)
                        .transition(.slideUp)
                    }
                }
                .animation(Anim.bouncy, value: vm.hasActiveRoute)
                .sensoryFeedback(.selection, trigger: selectedTab)
            }
        }
    }

    // MARK: - iPad Cockpit Rail Layout
    private var ipadCockpitLayout: some View {
        HStack(spacing: 0) {
            // Adaptive Rail
            VStack(spacing: LabTheme.s16) {
                // Tactical Driver Badge
                VStack(spacing: 6) {
                    Image(systemName: "steeringwheel")
                        .font(.system(size: 20, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(width: 44, height: 44)
                        .background(LabTheme.fg, in: Circle())

                    Text("DRIVER")
                        .font(.system(size: 9, weight: .black, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                }
                .padding(.top, 24)
                .padding(.bottom, 8)

                Divider()
                    .overlay(LabTheme.separator)
                    .padding(.horizontal, 12)

                // Navigation Items
                VStack(spacing: 12) {
                    ForEach(AppTab.allCases, id: \.self) { tab in
                        Button {
                            withAnimation(Anim.snappy) { selectedTab = tab }
                        } label: {
                            VStack(spacing: 4) {
                                Image(systemName: tab.icon)
                                    .font(.system(size: 18, weight: selectedTab == tab ? .bold : .regular))
                                Text(tab.label)
                                    .font(.system(size: 10, weight: selectedTab == tab ? .bold : .medium))
                            }
                            .foregroundStyle(selectedTab == tab ? LabTheme.fg : LabTheme.fgTertiary)
                            .frame(width: 64, height: 56)
                            .background(
                                RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                                    .fill(selectedTab == tab ? LabTheme.fg.opacity(0.08) : Color.clear)
                            )
                        }
                        .buttonStyle(.pressable)
                    }
                }

                Spacer()

                // Connectivity Dot
                VStack(spacing: 4) {
                    Circle()
                        .fill(wasOffline ? LabTheme.destructive : LabTheme.success)
                        .frame(width: 8, height: 8)
                    Text(wasOffline ? "OFFLINE" : "LIVE")
                        .font(.system(size: 8, weight: .black, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                }
                .padding(.bottom, 24)
            }
            .frame(width: 80)
            .background(LabTheme.card)
            .overlay(
                Rectangle()
                    .fill(LabTheme.separator)
                    .frame(width: 0.5),
                alignment: .trailing
            )

            // Main Cockpit Area
            ZStack(alignment: .bottom) {
                Group {
                    switch selectedTab {
                    case .home:
                        HomeView(vm: vm, onOpenMap: {
                            withAnimation(Anim.snappy) { selectedTab = .map }
                        }, onHandoffNavigate: { dest in
                            switch dest {
                            case .home: selectedTab = .home
                            case .fleetMap: selectedTab = .map
                            case .manifestList, .manifestDetail, .orderDetail: selectedTab = .rides
                            case .unresolved: break
                            }
                        })
                    case .map:
                        FleetMapView(vm: vm, goBack: {
                            withAnimation(Anim.snappy) { selectedTab = .home }
                        })
                    case .rides:
                        RidesListView(vm: vm)
                    case .profile:
                        ProfileView(vm: vm)
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)

                // Floating ActiveRideBar on iPad
                if selectedTab != .map && vm.hasActiveRoute, let mission = vm.activeMission {
                    ActiveRideBar(
                        mission: mission,
                        driverLocation: vm.location,
                        onTap: {
                            withAnimation(Anim.snappy) { selectedTab = .map }
                        }
                    )
                    .labReadableWidth()
                    .padding(.horizontal, LabTheme.s16)
                    .padding(.bottom, LabTheme.s16)
                    .transition(.slideUp)
                }
            }
        }
        .sensoryFeedback(.selection, trigger: selectedTab)
    }
}

#Preview {
    MainTabView()
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}
