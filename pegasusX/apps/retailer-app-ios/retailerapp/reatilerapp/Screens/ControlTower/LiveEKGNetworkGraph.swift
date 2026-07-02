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
    @State private var nodes: [NetworkNode] = []
    @State private var edges: [NetworkEdge] = []
    
    var body: some View {
        GeometryReader { geometry in
            TimelineView(.animation) { timeline in
                Canvas { context, size in
                    let now = timeline.date.timeIntervalSinceReferenceDate
                    
                    // Draw edges
                    for edge in edges {
                        var path = Path()
                        path.move(to: edge.from.position)
                        path.addQuadCurve(to: edge.to.position, control: edge.controlPoint)
                        
                        context.stroke(
                            path,
                            with: .color(.cyan.opacity(0.3)),
                            lineWidth: 2
                        )
                        
                        // Draw moving pulse along the curve
                        // t goes from 0 to 1 repeatedly
                        let duration = 2.0
                        let offset = Double(edge.id.hashValue) // pseudo-random offset
                        let t = (now + offset).truncatingRemainder(dividingBy: duration) / duration
                        
                        let pulsePos = quadraticBezier(t: t, p0: edge.from.position, p1: edge.controlPoint, p2: edge.to.position)
                        
                        let pulseRect = CGRect(x: pulsePos.x - 3, y: pulsePos.y - 3, width: 6, height: 6)
                        context.fill(Path(ellipseIn: pulseRect), with: .color(.white))
                        
                        // Add glow effect
                        let glowRect = CGRect(x: pulsePos.x - 6, y: pulsePos.y - 6, width: 12, height: 12)
                        context.fill(Path(ellipseIn: glowRect), with: .color(.cyan.opacity(0.8)))
                    }
                    
                    // Draw nodes
                    for node in nodes {
                        drawNode(node: node, in: &context)
                    }
                }
            }
            .onAppear {
                generateGraph(in: geometry.size)
            }
        }
        .background(Color.black)
    }
    
    private func generateGraph(in size: CGSize) {
        let padding: CGFloat = 40
        let w = size.width - padding * 2
        let h = size.height - padding * 2
        
        let warehouse = NetworkNode(position: CGPoint(x: size.width / 2, y: padding), type: .warehouse)
        let retailer1 = NetworkNode(position: CGPoint(x: padding, y: size.height - padding), type: .retailer)
        let retailer2 = NetworkNode(position: CGPoint(x: size.width - padding, y: size.height - padding), type: .retailer)
        let driver1 = NetworkNode(position: CGPoint(x: size.width * 0.3, y: size.height * 0.5), type: .driver)
        let driver2 = NetworkNode(position: CGPoint(x: size.width * 0.7, y: size.height * 0.5), type: .driver)
        
        nodes = [warehouse, retailer1, retailer2, driver1, driver2]
        
        edges = [
            NetworkEdge(from: warehouse, to: driver1, controlPoint: CGPoint(x: size.width * 0.2, y: size.height * 0.3)),
            NetworkEdge(from: driver1, to: retailer1, controlPoint: CGPoint(x: size.width * 0.2, y: size.height * 0.7)),
            NetworkEdge(from: warehouse, to: driver2, controlPoint: CGPoint(x: size.width * 0.8, y: size.height * 0.3)),
            NetworkEdge(from: driver2, to: retailer2, controlPoint: CGPoint(x: size.width * 0.8, y: size.height * 0.7))
        ]
    }
    
    private func quadraticBezier(t: Double, p0: CGPoint, p1: CGPoint, p2: CGPoint) -> CGPoint {
        let x = pow(1 - t, 2) * p0.x + 2 * (1 - t) * t * p1.x + pow(t, 2) * p2.x
        let y = pow(1 - t, 2) * p0.y + 2 * (1 - t) * t * p1.y + pow(t, 2) * p2.y
        return CGPoint(x: x, y: y)
    }
    
    private func drawNode(node: NetworkNode, in context: inout GraphicsContext) {
        let size: CGFloat = 16
        let rect = CGRect(x: node.position.x - size/2, y: node.position.y - size/2, width: size, height: size)
        
        switch node.type {
        case .warehouse:
            var path = Path()
            path.move(to: CGPoint(x: rect.midX, y: rect.minY))
            path.addLine(to: CGPoint(x: rect.maxX, y: rect.maxY))
            path.addLine(to: CGPoint(x: rect.minX, y: rect.maxY))
            path.closeSubpath()
            context.fill(path, with: .color(.orange))
        case .retailer:
            context.fill(Path(rect), with: .color(.blue))
        case .driver:
            context.fill(Path(ellipseIn: rect), with: .color(.green))
        }
    }
}

#Preview {
    LiveEKGNetworkGraph()
}
