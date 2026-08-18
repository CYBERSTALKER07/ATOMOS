import SwiftUI

struct SidePanel: View {
    @Binding var tab: PanelTab
    let selectedBuilding: String?
    let selectedTrace: String?
    let hovered: String?
    var onSelectBuilding: (String) -> Void
    var onSelectTrace: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 0) {
                tabBtn(.path, "WHAT IT DOES")
                tabBtn(.hall, "HOW IT'S BUILT")
                tabBtn(.key, "KEY")
            }
            Rectangle().fill(Ink.hair).frame(height: 1)
            Group {
                switch tab {
                case .path:
                    PathNote(trace: selectedTrace.flatMap(Atlas.trace))
                case .hall:
                    HallNote(
                        building: (hovered ?? selectedBuilding).flatMap(Atlas.building)
                    )
                case .key:
                    KeyNote()
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
        .background(Ink.bg)
        .foregroundStyle(Ink.fg)
    }

    private func tabBtn(_ item: PanelTab, _ title: String) -> some View {
        Button {
            tab = item
        } label: {
            Text(title)
                .font(Typeface.mono(8))
                .tracking(0.6)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 11)
                .foregroundStyle(tab == item ? Ink.invert : Ink.dim)
                .background(tab == item ? Ink.fg : Ink.bg)
        }
        .buttonStyle(.plain)
    }
}

struct PathNote: View {
    let trace: Trace?

    var body: some View {
        ScrollView {
            if let trace {
                VStack(alignment: .leading, spacing: 14) {
                    Text("SELECTED FLOW")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                    HStack {
                        Text(trace.title)
                            .font(Typeface.display(20, weight: .bold))
                        Spacer()
                        VerdictChip(trace.verdict)
                    }
                    Text(trace.explainer)
                        .font(Typeface.serif(14))
                        .lineSpacing(4)

                    Rectangle().fill(Ink.hair).frame(height: 1)

                    Text("HOPS")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                    Text(trace.hops.joined(separator: "  →  "))
                        .font(Typeface.serif(13))
                        .lineSpacing(3)

                    Text("PAYLOAD")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                    Text(trace.payload)
                        .font(Typeface.mono(11))
                        .padding(8)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .overlay(Rectangle().stroke(Ink.faint, lineWidth: 1))
                        .textSelection(.enabled)

                    Text("SOURCE")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                    CiteList(cites: Array(trace.cites.prefix(6)))
                }
                .padding(18)
            } else {
                Text("Choose a path in the header.")
                    .font(Typeface.serif(14))
                    .foregroundStyle(Ink.dim)
                    .padding(18)
            }
        }
    }
}

struct HallNote: View {
    let building: Building?

    var body: some View {
        ScrollView {
            if let building {
                VStack(alignment: .leading, spacing: 12) {
                    Text("SELECTED HALL")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                    HStack {
                        Text(building.code)
                            .font(Typeface.mono(14))
                        Text(building.name)
                            .font(Typeface.display(20, weight: .bold))
                        Spacer()
                        VerdictChip(building.verdict)
                    }
                    Text(building.district.rawValue.uppercased())
                        .font(Typeface.mono(9))
                        .foregroundStyle(Ink.dim)
                    Text(building.blurb)
                        .font(Typeface.serif(14))
                        .lineSpacing(4)
                    Rectangle().fill(Ink.hair).frame(height: 1)
                    Text("SOURCE")
                        .font(Typeface.mono(8))
                        .foregroundStyle(Ink.dim)
                    CiteList(cites: building.cites)
                }
                .padding(18)
            } else {
                Text("Select a hall from the left list or the grid.")
                    .font(Typeface.serif(14))
                    .foregroundStyle(Ink.dim)
                    .padding(18)
            }
        }
    }
}

struct KeyNote: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text("LINE")
                    .font(Typeface.mono(8))
                    .foregroundStyle(Ink.dim)
                ForEach(LaneKind.allCases) { lane in
                    HStack(alignment: .top, spacing: 10) {
                        DashSwatch(dash: lane.dash)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(lane.rawValue)
                                .font(Typeface.display(13, weight: .semibold))
                            Text(lane.caption)
                                .font(Typeface.serif(12))
                                .foregroundStyle(Ink.dim)
                        }
                    }
                }
                Text("STAMP")
                    .font(Typeface.mono(8))
                    .foregroundStyle(Ink.dim)
                    .padding(.top, 8)
                ForEach(Verdict.allCases) { v in
                    HStack(alignment: .top, spacing: 8) {
                        VerdictChip(v)
                        Text(v.gloss)
                            .font(Typeface.serif(12))
                            .foregroundStyle(Ink.dim)
                    }
                }
                Text("1–9 path   ← → hall   click a code on the grid")
                    .font(Typeface.mono(10))
                    .foregroundStyle(Ink.dim)
                    .padding(.top, 8)
            }
            .padding(18)
        }
    }
}

struct DashSwatch: View {
    let dash: [CGFloat]
    var body: some View {
        Canvas { ctx, size in
            var p = Path()
            p.move(to: CGPoint(x: 0, y: size.height / 2))
            p.addLine(to: CGPoint(x: size.width, y: size.height / 2))
            ctx.stroke(p, with: .color(Ink.fg), style: StrokeStyle(lineWidth: 1.4, dash: dash))
        }
        .frame(width: 36, height: 12)
        .padding(.top, 4)
    }
}

struct CiteList: View {
    let cites: [Cite]
    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            ForEach(cites) { c in
                VStack(alignment: .leading, spacing: 1) {
                    Text(c.label)
                        .font(Typeface.mono(10))
                        .textSelection(.enabled)
                    if !c.note.isEmpty {
                        Text(c.note)
                            .font(Typeface.serif(11))
                            .foregroundStyle(Ink.dim)
                    }
                }
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .overlay(Rectangle().stroke(Ink.faint, lineWidth: 1))
    }
}

struct VerdictChip: View {
    let verdict: Verdict
    init(_ verdict: Verdict) { self.verdict = verdict }

    var body: some View {
        Text(verdict.rawValue)
            .font(Typeface.mono(8))
            .foregroundStyle(fill ? Ink.invert : Ink.fg)
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(fill ? Ink.fg : Color.clear)
            .overlay(Rectangle().stroke(Ink.fg, lineWidth: 1))
    }

    private var fill: Bool { verdict == .real || verdict == .partial }
}
