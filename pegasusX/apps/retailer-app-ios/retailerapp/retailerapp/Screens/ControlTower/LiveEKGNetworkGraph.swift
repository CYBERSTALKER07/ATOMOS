import SwiftUI

struct NetworkNode: Identifiable {
    let id = UUID()
    var position: CGPoint
    let type: NodeType
    
    enum NodeType {
        case warehouse, retailer, driver
    }
}

struct NetworkEdge: Identifiable {
    let id = UUID()
    let from: NetworkNode
    let to: NetworkNode
    let controlPoint: CGPoint
}

struct LiveEKGNetworkGraph: View {
    var body: some View {
        ControlTowerView()
    }
}

#Preview {
    LiveEKGNetworkGraph()
}

