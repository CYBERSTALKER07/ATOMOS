import SwiftUI

struct CityCanvas: View {
    let selectedBuilding: String?
    let selectedTrace: String?
    let hovered: String?
    let pan: CGSize
    let zoom: CGFloat

    var body: some View {
        TimelineView(.animation(minimumInterval: 1.0 / 24.0, paused: false)) { timeline in
            Canvas { context, size in
                draw(context: context, size: size, t: timeline.date.timeIntervalSinceReferenceDate)
            }
            .accessibilityLabel("Isometric pegasusX system map")
        }
        .background(Ink.bg)
        .clipped()
    }

    private func origin(in size: CGSize) -> CGPoint {
        CGPoint(x: size.width * 0.50 + pan.width, y: size.height * 0.14 + pan.height)
    }

    private func draw(context: GraphicsContext, size: CGSize, t: TimeInterval) {
        var ctx = context
        let o = origin(in: size)
        ctx.translateBy(x: o.x, y: o.y)
        ctx.scaleBy(x: zoom, y: zoom)
        ctx.translateBy(x: -o.x, y: -o.y)

        drawGround(ctx, origin: o)
        if let tr = selectedTrace.flatMap(Atlas.trace) {
            drawTrace(ctx, tr, origin: o, t: t)
        }
        let sorted = Atlas.buildings.sorted { ($0.gx + $0.gy) < ($1.gx + $1.gy) }
        for b in sorted {
            drawBuilding(ctx, b, origin: o)
        }
    }

    private func drawGround(_ ctx: GraphicsContext, origin o: CGPoint) {
        let cols = 16
        let rows = 13
        for y in 0...rows {
            var row = Path()
            row.move(to: Iso.project(0, Double(y), 0, origin: o))
            row.addLine(to: Iso.project(Double(cols), Double(y), 0, origin: o))
            ctx.stroke(row, with: .color(Ink.hair), lineWidth: 1)
        }
        for x in 0...cols {
            var col = Path()
            col.move(to: Iso.project(Double(x), 0, 0, origin: o))
            col.addLine(to: Iso.project(Double(x), Double(rows), 0, origin: o))
            ctx.stroke(col, with: .color(Ink.hair), lineWidth: 1)
        }
        let frame = Iso.quad(
            Iso.project(0, 0, 0, origin: o),
            Iso.project(Double(cols), 0, 0, origin: o),
            Iso.project(Double(cols), Double(rows), 0, origin: o),
            Iso.project(0, Double(rows), 0, origin: o)
        )
        ctx.stroke(frame, with: .color(Ink.faint), lineWidth: 1.2)
    }

    private func drawBuilding(_ ctx: GraphicsContext, _ b: Building, origin o: CGPoint) {
        let onPath = buildingOnSelectedTrace(b.id)
        let lit = selectedBuilding == b.id || hovered == b.id
        let muted = b.verdict == .absent || b.verdict == .gated || b.verdict == .gone
        let top = Ink.gray(lit ? 0.48 : (onPath ? 0.38 : 0.28))
        let left = Ink.gray(lit ? 0.22 : 0.12)
        let right = Ink.gray(lit ? 0.34 : 0.20)

        switch b.kind {
        case .silo:
            drawSilo(ctx, b, origin: o, muted: muted, lit: lit)
        case .mast:
            drawMast(ctx, b, origin: o, lit: lit)
        case .lot:
            drawLot(ctx, b, origin: o, lit: lit)
        default:
            drawBox(ctx, b, origin: o, top: top, left: left, right: right, muted: muted, lit: lit)
            if b.kind == .plant { drawChimneys(ctx, b, origin: o) }
            if b.kind == .warehouse || b.kind == .dock {
                drawBand(ctx, b, origin: o)
            }
        }

        if lit {
            let halo = Iso.quad(
                Iso.project(b.gx - 0.1, b.gy - 0.1, 0, origin: o),
                Iso.project(b.gx + b.w + 0.1, b.gy - 0.1, 0, origin: o),
                Iso.project(b.gx + b.w + 0.1, b.gy + b.d + 0.1, 0, origin: o),
                Iso.project(b.gx - 0.1, b.gy + b.d + 0.1, 0, origin: o)
            )
            ctx.stroke(halo, with: .color(Ink.fg), lineWidth: 1.5)
        }

        let cap = Iso.project(b.gx + b.w / 2, b.gy + b.d / 2, b.h + 0.05, origin: o)
        let mark = Text(b.code)
            .font(Typeface.mono(9))
            .foregroundColor(Ink.fg)
        ctx.draw(ctx.resolve(mark), at: cap, anchor: .center)
    }

    private func drawBox(
        _ ctx: GraphicsContext, _ b: Building, origin o: CGPoint,
        top: Color, left: Color, right: Color, muted: Bool, lit: Bool
    ) {
        let x = b.gx, y = b.gy, w = b.w, d = b.d, h = b.h
        let p000 = Iso.project(x, y, 0, origin: o)
        let p010 = Iso.project(x, y + d, 0, origin: o)
        let p110 = Iso.project(x + w, y + d, 0, origin: o)
        let p001 = Iso.project(x, y, h, origin: o)
        let p101 = Iso.project(x + w, y, h, origin: o)
        let p011 = Iso.project(x, y + d, h, origin: o)
        let p111 = Iso.project(x + w, y + d, h, origin: o)
        let leftFace = Iso.quad(p010, p000, p001, p011)
        let rightFace = Iso.quad(p010, p110, p111, p011)
        let topFace = Iso.quad(p001, p101, p111, p011)

        if muted {
            ctx.stroke(leftFace, with: .color(Ink.faint), style: StrokeStyle(lineWidth: 1, dash: [3, 2]))
            ctx.stroke(rightFace, with: .color(Ink.faint), style: StrokeStyle(lineWidth: 1, dash: [3, 2]))
            ctx.stroke(topFace, with: .color(Ink.dim), style: StrokeStyle(lineWidth: 1, dash: [3, 2]))
        } else {
            ctx.fill(leftFace, with: .color(left))
            ctx.fill(rightFace, with: .color(right))
            ctx.fill(topFace, with: .color(top))
            ctx.stroke(leftFace, with: .color(Ink.bg), lineWidth: 0.8)
            ctx.stroke(rightFace, with: .color(Ink.bg), lineWidth: 0.8)
            ctx.stroke(topFace, with: .color(Ink.fg.opacity(lit ? 0.55 : 0.22)), lineWidth: 1)
            if b.verdict == .theatre {
                hatch(ctx, topFace)
            }
        }
    }

    private func hatch(_ ctx: GraphicsContext, _ face: Path) {
        let box = face.boundingRect
        var h = Path()
        var x = box.minX
        while x < box.maxX + box.height {
            h.move(to: CGPoint(x: x, y: box.minY))
            h.addLine(to: CGPoint(x: x - box.height, y: box.maxY))
            x += 5
        }
        ctx.drawLayer { layer in
            layer.clip(to: face)
            layer.stroke(h, with: .color(Ink.fg.opacity(0.35)), lineWidth: 1)
        }
    }

    private func drawSilo(_ ctx: GraphicsContext, _ b: Building, origin o: CGPoint, muted: Bool, lit: Bool) {
        let cx = b.gx + b.w / 2
        let cy = b.gy + b.d / 2
        let rx = b.w * 0.48
        let ry = b.d * 0.48
        for i in 0...6 {
            let z = b.h * Double(i) / 6
            var path = Path()
            for s in 0...24 {
                let ang = Double(s) / 24 * .pi * 2
                let pt = Iso.project(cx + cos(ang) * rx, cy + sin(ang) * ry, z, origin: o)
                if s == 0 { path.move(to: pt) } else { path.addLine(to: pt) }
            }
            ctx.stroke(path, with: .color(Ink.fg.opacity(muted ? 0.25 : (i == 6 ? 0.8 : 0.35))), lineWidth: i == 6 ? 1.4 : 0.8)
        }
        if lit {
            let cap = Iso.project(cx, cy, b.h, origin: o)
            ctx.stroke(Path(ellipseIn: CGRect(x: cap.x - 7, y: cap.y - 3, width: 14, height: 6)), with: .color(Ink.fg), lineWidth: 1)
        }
    }

    private func drawMast(_ ctx: GraphicsContext, _ b: Building, origin o: CGPoint, lit: Bool) {
        let base = Iso.project(b.gx + b.w / 2, b.gy + b.d / 2, 0, origin: o)
        let tip = Iso.project(b.gx + b.w / 2, b.gy + b.d / 2, b.h, origin: o)
        var pole = Path()
        pole.move(to: base)
        pole.addLine(to: tip)
        ctx.stroke(pole, with: .color(Ink.fg), lineWidth: lit ? 2 : 1.2)
        ctx.stroke(Path(ellipseIn: CGRect(x: tip.x - 8, y: tip.y - 4, width: 16, height: 8)), with: .color(Ink.fg), lineWidth: 1)
    }

    private func drawLot(_ ctx: GraphicsContext, _ b: Building, origin o: CGPoint, lit: Bool) {
        let plate = Iso.quad(
            Iso.project(b.gx, b.gy, 0.03, origin: o),
            Iso.project(b.gx + b.w, b.gy, 0.03, origin: o),
            Iso.project(b.gx + b.w, b.gy + b.d, 0.03, origin: o),
            Iso.project(b.gx, b.gy + b.d, 0.03, origin: o)
        )
        ctx.stroke(plate, with: .color(Ink.fg.opacity(lit ? 0.9 : 0.4)), style: StrokeStyle(lineWidth: 1, dash: [4, 3]))
    }

    private func drawChimneys(_ ctx: GraphicsContext, _ b: Building, origin o: CGPoint) {
        for (dx, dy) in [(0.18, 0.18), (0.52, 0.22)] as [(Double, Double)] {
            let c = Building(
                id: b.id + "-ch", name: "", kind: .tower, district: b.district,
                gx: b.gx + dx, gy: b.gy + dy, w: 0.12, d: 0.12, h: b.h + 0.55,
                verdict: b.verdict, blurb: "", cites: []
            )
            drawBox(ctx, c, origin: o, top: Ink.gray(0.4), left: Ink.gray(0.14), right: Ink.gray(0.24), muted: false, lit: false)
        }
    }

    private func drawBand(_ ctx: GraphicsContext, _ b: Building, origin o: CGPoint) {
        let band = Iso.quad(
            Iso.project(b.gx, b.gy + b.d, 0.35, origin: o),
            Iso.project(b.gx + b.w, b.gy + b.d, 0.35, origin: o),
            Iso.project(b.gx + b.w, b.gy + b.d, 0.5, origin: o),
            Iso.project(b.gx, b.gy + b.d, 0.5, origin: o)
        )
        ctx.fill(band, with: .color(Ink.fg.opacity(0.12)))
    }

    private func drawTrace(_ ctx: GraphicsContext, _ tr: Trace, origin o: CGPoint, t: TimeInterval) {
        guard tr.waypoints.count > 1 else { return }
        var path = Path()
        var pts: [CGPoint] = []
        for (i, wp) in tr.waypoints.enumerated() {
            let p = Iso.project(wp.0, wp.1, 0.18, origin: o)
            pts.append(p)
            if i == 0 { path.move(to: p) } else { path.addLine(to: p) }
        }
        ctx.stroke(
            path,
            with: .color(Ink.fg),
            style: StrokeStyle(lineWidth: 1.8, lineCap: .round, lineJoin: .round, dash: tr.lane.dash)
        )
        if let last = pts.last, pts.count >= 2 {
            let prev = pts[pts.count - 2]
            drawArrow(ctx, from: prev, to: last)
        }
        let segs = packetPolyline(tr.waypoints, origin: o)
        guard segs.total > 0 else { return }
        let u = t.truncatingRemainder(dividingBy: 4) / 4
        let pos = segs.point(at: u * segs.total)
        ctx.fill(Path(ellipseIn: CGRect(x: pos.x - 3.5, y: pos.y - 3.5, width: 7, height: 7)), with: .color(Ink.fg))
        ctx.stroke(Path(ellipseIn: CGRect(x: pos.x - 3.5, y: pos.y - 3.5, width: 7, height: 7)), with: .color(Ink.bg), lineWidth: 1)
    }

    private func drawArrow(_ ctx: GraphicsContext, from: CGPoint, to: CGPoint) {
        let dx = to.x - from.x
        let dy = to.y - from.y
        let len = hypot(dx, dy)
        guard len > 1 else { return }
        let ux = dx / len
        let uy = dy / len
        let tip = to
        let left = CGPoint(x: tip.x - ux * 8 + uy * 4, y: tip.y - uy * 8 - ux * 4)
        let right = CGPoint(x: tip.x - ux * 8 - uy * 4, y: tip.y - uy * 8 + ux * 4)
        var p = Path()
        p.move(to: left)
        p.addLine(to: tip)
        p.addLine(to: right)
        ctx.stroke(p, with: .color(Ink.fg), lineWidth: 1.4)
    }

    private func buildingOnSelectedTrace(_ id: String) -> Bool {
        guard let tr = selectedTrace.flatMap(Atlas.trace), let b = Atlas.building(id) else { return false }
        for wp in tr.waypoints {
            if hypot(b.gx + b.w / 2 - wp.0, b.gy + b.d / 2 - wp.1) < 1.4 { return true }
        }
        return false
    }
}

private struct Polyline {
    let points: [CGPoint]
    let total: CGFloat
    func point(at distance: CGFloat) -> CGPoint {
        guard points.count > 1 else { return points.first ?? .zero }
        var remain = distance
        for i in 0..<(points.count - 1) {
            let a = points[i], b = points[i + 1]
            let len = hypot(b.x - a.x, b.y - a.y)
            if remain <= len {
                let u = len == 0 ? 0 : remain / len
                return CGPoint(x: a.x + (b.x - a.x) * u, y: a.y + (b.y - a.y) * u)
            }
            remain -= len
        }
        return points.last ?? .zero
    }
}

private func packetPolyline(_ wps: [(Double, Double)], origin o: CGPoint) -> Polyline {
    let pts = wps.map { Iso.project($0.0, $0.1, 0.4, origin: o) }
    var total: CGFloat = 0
    if pts.count > 1 {
        for i in 0..<(pts.count - 1) {
            total += hypot(pts[i + 1].x - pts[i].x, pts[i + 1].y - pts[i].y)
        }
    }
    return Polyline(points: pts, total: total)
}
