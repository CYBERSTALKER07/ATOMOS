//
//  MainTabView.swift
//  driverappios
//

import SwiftUI
import SwiftData

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
    @State private var selectedTab: AppTab = .home
    @State private var vm = FleetViewModel()

    var body: some View {
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

#Preview {
    MainTabView()
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}
