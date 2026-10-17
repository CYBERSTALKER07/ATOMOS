import SwiftUI

struct ContentView: View {
    @State private var selectedBuilding: String? = "order-hall"
    @State private var selectedTrace: String? = "order-create"
    @State private var hovered: String?
    @State private var pan: CGSize = .zero
    @State private var zoom: CGFloat = 1.0
    @State private var tab: PanelTab = .path

    var body: some View {
        VStack(spacing: 0) {
            header
            hair
            HStack(spacing: 0) {
                ModuleRail(
                    selectedBuilding: selectedBuilding,
                    onSelect: { id in
                        selectedBuilding = id
                        tab = .hall
                    }
                )
                .frame(width: 228)
                hairV
                center
                hairV
                SidePanel(
                    tab: $tab,
                    selectedBuilding: selectedBuilding,
                    selectedTrace: selectedTrace,
                    hovered: hovered,
                    onSelectBuilding: { id in
                        selectedBuilding = id
                        tab = .hall
                    },
                    onSelectTrace: { id in
                        selectedTrace = id
                        tab = .path
                    }
                )
                .frame(width: 340)
            }
            .frame(maxHeight: .infinity)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Ink.bg)
        .preferredColorScheme(.dark)
        .focusable()
        .onKeyPress { handleKey($0) }
    }

    private var header: some View {
        HStack(spacing: 0) {
            headerCell("REPOSITORY", Atlas.repo, Atlas.branch + "  ·  " + Atlas.commit, width: 280)
            headerCell("PATHS", "\(Atlas.traces.count)", nil, width: 88)
            headerCell("HALLS", "\(Atlas.buildings.count)", nil, width: 88)
            headerCell("SELECTED", selectedTrace.flatMap(Atlas.trace)?.title ?? "None", nil, width: 180)
            VStack(alignment: .leading, spacing: 4) {
                Text("PATH")
                    .font(Typeface.mono(8))
                    .foregroundStyle(Ink.dim)
                Menu {
                    ForEach(Atlas.traces) { tr in
                        Button(tr.title) {
                            selectedTrace = tr.id
                            tab = .path
                        }
                    }
                } label: {
                    HStack(spacing: 8) {
                        Text(selectedTrace.flatMap(Atlas.trace)?.title ?? "Choose")
                            .font(Typeface.display(13, weight: .semibold))
                            .foregroundStyle(Ink.fg)
                        Text("▾")
                            .font(Typeface.mono(10))
                            .foregroundStyle(Ink.dim)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .overlay(Rectangle().stroke(Ink.fg.opacity(0.45), lineWidth: 1))
                }
                .menuStyle(.borderlessButton)
                .tint(Ink.fg)
            }
            .padding(.horizontal, 14)
            .frame(maxHeight: .infinity, alignment: .leading)
            Spacer(minLength: 0)
        }
        .frame(height: 72)
        .background(Ink.bg)
    }

    private func headerCell(_ k: String, _ v: String, _ sub: String?, width: CGFloat) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(k)
                .font(Typeface.mono(8))
                .foregroundStyle(Ink.dim)
            Text(v)
                .font(Typeface.display(14, weight: .semibold))
                .lineLimit(1)
            if let sub {
                Text(sub)
                    .font(Typeface.mono(9))
                    .foregroundStyle(Ink.dim)
                    .lineLimit(1)
            }
        }
        .padding(.horizontal, 14)
        .frame(width: width, alignment: .leading)
        .frame(maxHeight: .infinity, alignment: .leading)
        .overlay(alignment: .trailing) { hairV }
    }



    private var center: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("RUNTIME TOPOLOGY")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                    Text(selectedTrace.flatMap(Atlas.trace)?.title ?? "All halls")
                        .font(Typeface.display(20, weight: .bold))
                }
                Spacer()
                HStack(spacing: 6) {
                    Circle().fill(Ink.fg).frame(width: 6, height: 6)
                    Text("PAYLOAD IN MOTION")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                }
            }
            .padding(.horizontal, 18)
            .padding(.vertical, 12)
            ZStack(alignment: .bottomLeading) {
                CityStage(
                    selectedBuilding: selectedBuilding,
                    selectedTrace: selectedTrace,
                    hovered: hovered,
                    pan: $pan,
                    zoom: $zoom,
                    onSelectBuilding: { id in
                        guard !id.isEmpty else { return }
                        selectedBuilding = id
                        tab = .hall
                    },
                    onHover: { hovered = $0 }
                )
                HStack(spacing: 14) {
                    legendLine("flow", [])
                    legendLine("event", [10, 5])
                    legendLine("money", [2, 4])
                    legendLine("theatre", [1, 5])
                    HStack(spacing: 5) {
                        Circle().fill(Ink.fg).frame(width: 6, height: 6)
                        Text("payload")
                            .font(Typeface.mono(9))
                            .foregroundStyle(Ink.dim)
                    }
                }
                .padding(14)
            }
        }
    }

    private func legendLine(_ name: String, _ dash: [CGFloat]) -> some View {
        HStack(spacing: 6) {
            DashSwatch(dash: dash).frame(width: 22, height: 8)
            Text(name)
                .font(Typeface.mono(9))
                .foregroundStyle(Ink.dim)
        }
    }

    private var hair: some View { Rectangle().fill(Ink.hair).frame(height: 1) }
    private var hairV: some View { Rectangle().fill(Ink.hair).frame(width: 1) }

    private func handleKey(_ press: KeyPress) -> KeyPress.Result {
        switch press.key {
        case .escape:
            selectedTrace = nil
            return .handled
        case .leftArrow, .rightArrow:
            cycle(Atlas.buildings.map(\.id), current: &selectedBuilding, forward: press.key == .rightArrow)
            tab = .hall
            return .handled
        case .upArrow, .downArrow:
            cycle(Atlas.traces.map(\.id), current: &selectedTrace, forward: press.key == .downArrow)
            tab = .path
            return .handled
        default:
            if let n = Int(press.characters), n >= 1, n <= Atlas.traces.count {
                selectedTrace = Atlas.traces[n - 1].id
                tab = .path
                return .handled
            }
            return .ignored
        }
    }

    private func cycle(_ ids: [String], current: inout String?, forward: Bool) {
        guard !ids.isEmpty else { return }
        let idx = current.flatMap { ids.firstIndex(of: $0) } ?? 0
        current = ids[forward ? (idx + 1) % ids.count : (idx - 1 + ids.count) % ids.count]
    }
}

struct ModuleRail: View {
    let selectedBuilding: String?
    var onSelect: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("THE SYSTEM")
                .font(Typeface.mono(8))
                .foregroundStyle(Ink.dim)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
            Rectangle().fill(Ink.hair).frame(height: 1)
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                ForEach(District.allCases) { d in
                    let members = Atlas.buildings.filter { $0.district == d }
                    if !members.isEmpty {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(d.rawValue.uppercased())
                                .font(Typeface.mono(8))
                                .foregroundStyle(Ink.dim)
                                .padding(.horizontal, 14)
                                .padding(.top, 4)
                            ForEach(members) { b in
                                Button {
                                    onSelect(b.id)
                                } label: {
                                    HStack(spacing: 8) {
                                        Text(b.code)
                                            .font(Typeface.mono(10))
                                            .frame(width: 22, alignment: .leading)
                                        Text(b.name)
                                            .font(Typeface.display(12, weight: .regular))
                                            .lineLimit(1)
                                        Spacer(minLength: 4)
                                        Text(shortVerdict(b.verdict))
                                            .font(Typeface.mono(8))
                                            .foregroundStyle(Ink.dim)
                                    }
                                    .foregroundStyle(Ink.fg)
                                    .padding(.horizontal, 14)
                                    .padding(.vertical, 6)
                                    .background(selectedBuilding == b.id ? Ink.fg.opacity(0.10) : Color.clear)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
                }
                .padding(.vertical, 12)
            }
        }
        .background(Ink.bg)
    }

    private func shortVerdict(_ v: Verdict) -> String {
        switch v {
        case .real: return "R"
        case .partial: return "P"
        case .theatre: return "T"
        case .gone: return "G"
        case .gated: return "X"
        case .absent: return "—"
        }
    }
}

struct CityStage: View {
    let selectedBuilding: String?
    let selectedTrace: String?
    let hovered: String?
    @Binding var pan: CGSize
    @Binding var zoom: CGFloat
    var onSelectBuilding: (String) -> Void
    var onHover: (String?) -> Void

    var body: some View {
        GeometryReader { geo in
            ZStack {
                CityCanvas(
                    selectedBuilding: selectedBuilding,
                    selectedTrace: selectedTrace,
                    hovered: hovered,
                    pan: pan,
                    zoom: zoom
                )
                Color.clear
                    .contentShape(Rectangle())
                    .simultaneousGesture(
                        MagnifyGesture().onChanged { zoom = min(1.6, max(0.7, $0.magnification)) }
                    )
                    .gesture(
                        DragGesture(minimumDistance: 0)
                            .onChanged { onHover(hit($0.location, size: geo.size)) }
                            .onEnded { value in
                                let moved = hypot(value.translation.width, value.translation.height)
                                if moved < 5 {
                                    if let id = hit(value.location, size: geo.size) { onSelectBuilding(id) }
                                } else if hit(value.startLocation, size: geo.size) == nil {
                                    pan.width += value.translation.width
                                    pan.height += value.translation.height
                                }
                            }
                    )
            }
        }
    }

    private func hit(_ p: CGPoint, size: CGSize) -> String? {
        let o = CGPoint(x: size.width * 0.50 + pan.width, y: size.height * 0.14 + pan.height)
        let qx = o.x + (p.x - o.x) / max(zoom, 0.01)
        let qy = o.y + (p.y - o.y) / max(zoom, 0.01)
        var best: (String, CGFloat)?
        for b in Atlas.buildings {
            let c = Iso.project(b.gx + b.w / 2, b.gy + b.d / 2, b.h * 0.5, origin: o)
            let dist = hypot(c.x - qx, c.y - qy)
            if dist < 28, best == nil || dist < best!.1 { best = (b.id, dist) }
        }
        return best?.0
    }
}
