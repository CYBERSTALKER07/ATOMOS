import SwiftUI

struct ControlTowerView: View {
    @State private var selectedTab = 0
    
    // Sample Data
    let nodes = [
        NetworkNode(id: "WH1", type: "warehouse", label: "Central Hub"),
        NetworkNode(id: "R1", type: "retailer", label: "Retailer A"),
        NetworkNode(id: "R2", type: "retailer", label: "Retailer B"),
        NetworkNode(id: "D1", type: "driver", label: "Driver 1"),
        NetworkNode(id: "D2", type: "driver", label: "Driver 2")
    ]
    
    let links = [
        NetworkLink(sourceId: "WH1", targetId: "R1"),
        NetworkLink(sourceId: "WH1", targetId: "D1"),
        NetworkLink(sourceId: "D1", targetId: "R2"),
        NetworkLink(sourceId: "WH1", targetId: "R2"),
        NetworkLink(sourceId: "WH1", targetId: "D2")
    ]

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                // Header
                HStack {
                    Text("Control Tower")
                        .font(.largeTitle)
                        .bold()
                    Spacer()
                    Image(systemName: "antenna.radiowaves.left.and.right")
                        .foregroundColor(.green)
                }
                .padding(.horizontal)
                .padding(.top)

                // Segmented Picker for Views
                Picker("View", selection: $selectedTab) {
                    Text("Live Network").tag(0)
                    Text("H3 Density Map").tag(1)
                }
                .pickerStyle(SegmentedPickerStyle())
                .padding(.horizontal)

                // Main Display Area
                ZStack {
                    RoundedRectangle(cornerRadius: 16)
                        .fill(Color(hex: "#1F2937"))
                        .shadow(radius: 5)
                    
                    if selectedTab == 0 {
                        LiveEKGNetworkGraph(nodes: nodes, links: links)
                            .clipShape(RoundedRectangle(cornerRadius: 16))
                    } else {
                        HexagonalControlTowerMap()
                            .clipShape(RoundedRectangle(cornerRadius: 16))
                    }
                }
                .frame(height: 400)
                .padding(.horizontal)

                // Metrics / Quick Stats
                HStack(spacing: 16) {
                    MetricCard(title: "Active Routes", value: "24", icon: "point.topleft.down.curvedto.point.bottomright.up")
                    MetricCard(title: "Alerts", value: "2", icon: "exclamationmark.triangle", color: .yellow)
                    MetricCard(title: "Efficiency", value: "98%", icon: "bolt.fill", color: .green)
                }
                .padding(.horizontal)
                
                Spacer(minLength: 40)
            }
        }
        .background(Color(hex: "#111827").edgesIgnoringSafeArea(.all))
        .navigationTitle("Control Tower")
        .navigationBarTitleDisplayMode(.inline)
    }
}

struct MetricCard: View {
    let title: String
    let value: String
    let icon: String
    var color: Color = .blue
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Image(systemName: icon)
                .foregroundColor(color)
                .font(.title2)
            Text(value)
                .font(.title)
                .bold()
                .foregroundColor(.white)
            Text(title)
                .font(.caption)
                .foregroundColor(.gray)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(Color(hex: "#1F2937"))
        .cornerRadius(12)
    }
}
