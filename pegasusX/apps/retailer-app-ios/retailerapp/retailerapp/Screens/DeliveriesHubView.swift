import SwiftUI

enum DeliveriesHubTab: String, CaseIterable, Identifiable {
    case map
    case dock

    var id: String { rawValue }

    var title: String {
        switch self {
        case .map: "Map"
        case .dock: "Dock Queue"
        }
    }
}

struct DeliveriesHubView: View {
    var initialTab: DeliveriesHubTab = .map

    @State private var selectedTab: DeliveriesHubTab

    init(initialTab: DeliveriesHubTab = .map) {
        self.initialTab = initialTab
        _selectedTab = State(initialValue: initialTab)
    }

    var body: some View {
        VStack(spacing: 0) {
            Picker("Deliveries", selection: $selectedTab) {
                ForEach(DeliveriesHubTab.allCases) { tab in
                    Text(tab.title).tag(tab)
                }
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingSM)

            switch selectedTab {
            case .map:
                DeliveryMapView()
            case .dock:
                DockView()
            }
        }
        .background(AppTheme.background)
        .navigationTitle("mobile_retailer.ui.deliveries")
        .navigationBarTitleDisplayMode(.inline)
        .onChange(of: initialTab) { _, tab in
            selectedTab = tab
        }
    }
}

#Preview {
    NavigationStack {
        DeliveriesHubView()
    }
}
