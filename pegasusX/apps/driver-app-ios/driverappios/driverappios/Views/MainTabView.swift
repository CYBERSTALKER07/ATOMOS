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
    @State private var selectedTab: AppTab = .home
    @State private var vm = FleetViewModel()
    @State private var refreshEpoch: Int = 0
    @State private var pathMonitor: NWPathMonitor?
    @State private var wasOffline = false

    var body: some View {
        Group {
            if selectedTab == .map {
                FleetMapView(vm: vm, goBack: {
                    withAnimation(Anim.snappy) { selectedTab = .home }
                })
                .id("map-\(refreshEpoch)")
            } else {
                VStack(spacing: 0) {
                    TabView(selection: $selectedTab) {
                        Tab("Home", systemImage: "house.fill", value: .home) {
                            HomeView(vm: vm, onOpenMap: {
                                withAnimation(Anim.snappy) { selectedTab = .map }
                            })
                            .id("home-\(refreshEpoch)")
                        }

                        Tab("Rides", systemImage: "list.bullet", value: .rides) {
                            RidesListView(vm: vm)
                                .id("rides-\(refreshEpoch)")
                        }

                        Tab("Profile", systemImage: "person.fill", value: .profile) {
                            ProfileView(vm: vm)
                                .id("profile-\(refreshEpoch)")
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
                        .padding(.horizontal, LabTheme.s16)
                        .padding(.bottom, LabTheme.s8)
                        .transition(.slideUp)
                    }
                }
                .animation(Anim.bouncy, value: vm.hasActiveRoute)
                .sensoryFeedback(.selection, trigger: selectedTab)
            }
        }
        .onChange(of: scenePhase) { _, newPhase in
            if newPhase == .active {
                refreshEpoch += 1
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
                            refreshEpoch += 1
                            Task {
                                await FleetServiceLive.shared.flushOfflineQueue()
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
    }
}

#Preview {
    MainTabView()
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}
