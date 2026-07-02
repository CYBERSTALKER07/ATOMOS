import SwiftUI

struct NetworkNode: Identifiable {
    let id: String
    let type: String // "warehouse", "retailer", "driver"
    let label: String
    var position: CGPoint = .zero
}

struct NetworkLink: Identifiable {
    let id = UUID()
    let sourceId: String
    let targetId: String
}

struct LiveEKGNetworkGraph: View {
    @State var nodes: [NetworkNode]
    let links: [NetworkLink]
    
    @State private var pulsePhase: CGFloat = 0.0

    var body: some View {
        GeometryReader { geometry in
            ZStack {
                Color.black.edgesIgnoringSafeArea(.all)
                
                let width = geometry.size.width
                let height = geometry.size.height
                let center = CGPoint(x: width / 2, y: height / 2)
                let radius = min(width, height) * 0.35
                
                let layoutNodes = computeLayout(nodes: nodes, center: center, radius: radius)
                
                // Draw Links
                ForEach(links) { link in
                    if let source = layoutNodes.first(where: { $0.id == link.sourceId }),
                       let target = layoutNodes.first(where: { $0.id == link.targetId }) {
                        
                        let dx = target.position.x - source.position.x
                        let dy = target.position.y - source.position.y
                        let controlPoint = CGPoint(
                            x: source.position.x + dx / 2 - dy * 0.2,
                            y: source.position.y + dy / 2 + dx * 0.2
                        )
                        
                        Path { path in
                            path.move(to: source.position)
                            path.addQuadCurve(to: target.position, control: controlPoint)
                        }
                        .stroke(Color(hex: "#4B5563"), lineWidth: 3)
                        
                        // Pulse Point
                        let pulsePt = getQuadraticBezierPoint(t: pulsePhase, p0: source.position, p1: controlPoint, p2: target.position)
                        Circle()
                            .fill(Color(hex: "#10B981").opacity(0.8))
                            .frame(width: 16, height: 16)
                            .position(pulsePt)
                    }
                }
                
                // Draw Nodes
                ForEach(layoutNodes) { node in
                    nodeView(for: node)
                        .position(node.position)
                }
            }
        }
        .onAppear {
            withAnimation(Animation.linear(duration: 2.0).repeatForever(autoreverses: false)) {
                pulsePhase = 1.0
            }
        }
    }
    
    private func computeLayout(nodes: [NetworkNode], center: CGPoint, radius: CGFloat) -> [NetworkNode] {
        var newNodes = nodes
        for i in 0..<newNodes.count {
            if newNodes[i].type == "warehouse" {
                newNodes[i].position = center
            } else {
                let angle = CGFloat(i) * 2.0 * .pi / CGFloat(newNodes.count)
                newNodes[i].position = CGPoint(
                    x: center.x + radius * cos(angle),
                    y: center.y + radius * sin(angle)
                )
            }
        }
        return newNodes
    }
    
    @ViewBuilder
    private func nodeView(for node: NetworkNode) -> some View {
        let color = colorFor(type: node.type)
        if node.type == "warehouse" {
            Triangle()
                .fill(color)
                .frame(width: 30, height: 30)
        } else if node.type == "retailer" {
            Rectangle()
                .fill(color)
                .frame(width: 30, height: 30)
        } else {
            Circle()
                .fill(color)
                .frame(width: 30, height: 30)
        }
    }
    
    private func colorFor(type: String) -> Color {
        switch type {
        case "warehouse": return Color(hex: "#10B981")
        case "retailer": return Color(hex: "#3B82F6")
        default: return Color(hex: "#F59E0B")
        }
    }
    
    private func getQuadraticBezierPoint(t: CGFloat, p0: CGPoint, p1: CGPoint, p2: CGPoint) -> CGPoint {
        let x = (1 - t) * (1 - t) * p0.x + 2 * (1 - t) * t * p1.x + t * t * p2.x
        let y = (1 - t) * (1 - t) * p0.y + 2 * (1 - t) * t * p1.y + t * t * p2.y
        return CGPoint(x: x, y: y)
    }
}

struct Triangle: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()
        path.move(to: CGPoint(x: rect.midX, y: rect.minY))
        path.addLine(to: CGPoint(x: rect.minX, y: rect.maxY))
        path.addLine(to: CGPoint(x: rect.maxX, y: rect.maxY))
        path.closeSubpath()
        return path
    }
}

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let a, r, g, b: UInt64
        switch hex.count {
        case 3: // RGB (12-bit)
            (a, r, g, b) = (255, (int >> 8) * 17, (int >> 4 & 0xF) * 17, (int & 0xF) * 17)
        case 6: // RGB (24-bit)
            (a, r, g, b) = (255, int >> 16, int >> 8 & 0xFF, int & 0xFF)
        case 8: // ARGB (32-bit)
            (a, r, g, b) = (int >> 24, int >> 16 & 0xFF, int >> 8 & 0xFF, int & 0xFF)
        default:
            (a, r, g, b) = (1, 1, 1, 0)
        }
        self.init(
            .sRGB,
            red: Double(r) / 255,
            green: Double(g) / 255,
            blue:  Double(b) / 255,
            opacity: Double(a) / 255
        )
    }
}
