import SwiftUI

private enum DeliveriesHubTab: String, CaseIterable, Identifiable {
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
    @State private var selectedTab: DeliveriesHubTab = .map

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
        .navigationTitle("Deliveries")
        .navigationBarTitleDisplayMode(.inline)
    }
}

#Preview {
    NavigationStack {
        DeliveriesHubView()
    }
}
