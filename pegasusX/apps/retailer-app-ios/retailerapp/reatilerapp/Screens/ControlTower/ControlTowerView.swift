import SwiftUI
import Charts

struct ControlTowerView: View {
    @State private var selectedView: ViewMode = .network
    
    enum ViewMode {
        case network
        case spatial
    }
    
    var body: some View {
        ZStack {
            // Background View
            if selectedView == .network {
                LiveEKGNetworkGraph()
                    .edgesIgnoringSafeArea(.all)
            } else {
                HexagonalControlTowerMap()
                    .edgesIgnoringSafeArea(.all)
            }
            
            // Overlay UI
            VStack {
                // Top Bar Segmented Control
                Picker("View Mode", selection: $selectedView) {
                    Text("EKG Network").tag(ViewMode.network)
                    Text("Spatial Map").tag(ViewMode.spatial)
                }
                .pickerStyle(SegmentedPickerStyle())
                .padding()
                .background(.ultraThinMaterial)
                .cornerRadius(12)
                .padding()
                
                Spacer()
                
                // Analytics Panel
                VStack(alignment: .leading, spacing: 16) {
                    Text("Live Pulse Analytics")
                        .font(.headline)
                        .foregroundColor(.white)
                    
                    HStack(spacing: 20) {
                        Chart {
                            BarMark(x: .value("Day", "Mon"), y: .value("Orders", 150))
                            BarMark(x: .value("Day", "Tue"), y: .value("Orders", 200))
                            BarMark(x: .value("Day", "Wed"), y: .value("Orders", 180))
                        }
                        .frame(height: 100)
                        
                        Chart {
                            LineMark(x: .value("Time", "10am"), y: .value("Load", 40))
                            LineMark(x: .value("Time", "12pm"), y: .value("Load", 85))
                            LineMark(x: .value("Time", "2pm"), y: .value("Load", 60))
                        }
                        .frame(height: 100)
                        .foregroundStyle(.orange)
                    }
                }
                .padding()
                .background(.ultraThinMaterial)
                .cornerRadius(16)
                .padding()
            }
        }
        .preferredColorScheme(.dark)
    }
}

#Preview {
    ControlTowerView()
}
