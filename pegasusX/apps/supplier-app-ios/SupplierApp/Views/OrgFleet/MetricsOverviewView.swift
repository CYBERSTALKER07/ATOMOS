import SwiftUI

struct MetricsOverviewView: View {
    let driversCount: Int
    let vehiclesCount: Int
    let orgMembersCount: Int
    let topology: SupplierTopologyResponse?
    
    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 16) {
                MetricCard(title: "mobile_supplier.chrome.warehouses", count: topology?.warehouses.count ?? 0)
                MetricCard(title: "mobile_supplier.chrome.factories", count: topology?.factories.count ?? 0)
                MetricCard(title: "mobile_supplier.chrome.org_members", count: orgMembersCount)
                MetricCard(title: "mobile_supplier.chrome.fleet_entities", count: driversCount + vehiclesCount)
            }
            .padding(.horizontal)
        }
    }
}

private struct MetricCard: View {
    let title: String
    let count: Int
    
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(LocalizedStringKey(title))
                .font(.caption)
                .foregroundStyle(.secondary)
            Text("\(count)")
                .font(.title2)
                .bold()
        }
        .padding()
        .frame(minWidth: 130, alignment: .leading)
        .background(Color(uiColor: .secondarySystemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}
