import SwiftUI

public struct SidebarView: View {
    @Binding public var selectedCategory: String
    @Binding public var selectedStatus: String
    public let requests: [EnterpriseRequest]

    public var body: some View {
        VStack(spacing: 0) {
            // Header with Pegasus swift emblem
            HStack(spacing: 12) {
                SwissCADTheme.logoImage
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(width: 32, height: 32)
                    .padding(4)
                    .background(SwissCADTheme.cardDark)
                    .overlay(
                        Rectangle()
                            .stroke(SwissCADTheme.borderStroke, lineWidth: 1)
                    )

                VStack(alignment: .leading, spacing: 2) {
                    HStack(spacing: 4) {
                        Rectangle()
                            .fill(SwissCADTheme.emeraldGreen)
                            .frame(width: 6, height: 6)
                        Text("PEGASUS // OS")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundColor(SwissCADTheme.electricCyan)
                    }
                    Text("Operator Studio")
                        .font(.system(size: 11, weight: .light))
                        .foregroundColor(Color.white.opacity(0.8))
                }
                Spacer()
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
            .background(SwissCADTheme.panelDark)
            .overlay(Rectangle().frame(height: 1).foregroundColor(SwissCADTheme.borderStroke), alignment: .bottom)

            List {
                Section(header: Text("CATEGORIES").font(.system(size: 10, weight: .bold, design: .monospaced)).foregroundColor(SwissCADTheme.techGray)) {
                Button(action: { selectedCategory = "all" }) {
                    HStack {
                        Image(systemName: "square.grid.2x2")
                        Text("All Inquiries")
                        Spacer()
                        Text("\(requests.count)")
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(SwissCADTheme.techGray)
                    }
                }
                .buttonStyle(.plain)
                .listRowBackground(selectedCategory == "all" ? SwissCADTheme.cardDark : Color.clear)

                Button(action: { selectedCategory = "demo" }) {
                    HStack {
                        Image(systemName: "shippingbox")
                        Text("Demo Requests")
                        Spacer()
                        Text("\(requests.filter { $0.inquiry_type == "demo" }.count)")
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(SwissCADTheme.techGray)
                    }
                }
                .buttonStyle(.plain)
                .listRowBackground(selectedCategory == "demo" ? SwissCADTheme.cardDark : Color.clear)

                Button(action: { selectedCategory = "pilot" }) {
                    HStack {
                        Image(systemName: "network")
                        Text("Pilot Programs")
                        Spacer()
                        Text("\(requests.filter { $0.inquiry_type == "pilot" }.count)")
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(SwissCADTheme.techGray)
                    }
                }
                .buttonStyle(.plain)
                .listRowBackground(selectedCategory == "pilot" ? SwissCADTheme.cardDark : Color.clear)

                Button(action: { selectedCategory = "general" }) {
                    HStack {
                        Image(systemName: "envelope")
                        Text("General & Contact")
                        Spacer()
                        Text("\(requests.filter { $0.inquiry_type == "general" }.count)")
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundColor(SwissCADTheme.techGray)
                    }
                }
                .buttonStyle(.plain)
                .listRowBackground(selectedCategory == "general" ? SwissCADTheme.cardDark : Color.clear)
            }

            Section(header: Text("WORKFLOW STATUS").font(.system(size: 10, weight: .bold, design: .monospaced)).foregroundColor(SwissCADTheme.techGray)) {
                ForEach(["all", "new", "in_review", "contacted", "qualified", "closed"], id: \.self) { status in
                    Button(action: { selectedStatus = status }) {
                        HStack {
                            Circle()
                                .fill(SwissCADTheme.statusColor(for: status))
                                .frame(width: 8, height: 8)
                            Text(status.replacingOccurrences(of: "_", with: " ").capitalized)
                            Spacer()
                            let count = status == "all" ? requests.count : requests.filter { $0.status == status }.count
                            Text("\(count)")
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundColor(SwissCADTheme.techGray)
                        }
                    }
                    .buttonStyle(.plain)
                    .listRowBackground(selectedStatus == status ? SwissCADTheme.cardDark : Color.clear)
                }
            }
        }
        .listStyle(.sidebar)
        }
        .background(SwissCADTheme.panelDark)
    }
}
