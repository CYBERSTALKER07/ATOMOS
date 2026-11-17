import SwiftUI

enum Iso {
    static let tileW: CGFloat = 38
    static let tileH: CGFloat = 19
    static let tileZ: CGFloat = 16

    static func project(_ x: Double, _ y: Double, _ z: Double, origin: CGPoint) -> CGPoint {
        CGPoint(
            x: origin.x + CGFloat(x - y) * tileW,
            y: origin.y + CGFloat(x + y) * tileH - CGFloat(z) * tileZ
        )
    }

    static func path(_ points: [CGPoint]) -> Path {
        var p = Path()
        guard let first = points.first else { return p }
        p.move(to: first)
        for pt in points.dropFirst() { p.addLine(to: pt) }
        p.closeSubpath()
        return p
    }

    static func quad(_ a: CGPoint, _ b: CGPoint, _ c: CGPoint, _ d: CGPoint) -> Path {
        path([a, b, c, d])
    }
}

struct HitBox {
    let id: String
    let path: Path
}
