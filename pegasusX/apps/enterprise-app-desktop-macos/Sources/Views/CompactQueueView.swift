import SwiftUI

public struct CompactQueueView: View {
    @Binding public var requests: [EnterpriseRequest]
    @Binding public var selectedCategory: String
    @Binding public var selectedStatus: String
    @Binding public var searchQuery: String
    public let onRefresh: () async -> Void
    public let onStatusChange: (String, String) -> Void
    public let onDelete: (String) -> Void
    public let onAnswerSent: () -> Void

    @State private var activeDetailRequest: EnterpriseRequest? = nil

    public init(
        requests: Binding<[EnterpriseRequest]>,
        selectedCategory: Binding<String>,
        selectedStatus: Binding<String>,
        searchQuery: Binding<String>,
        onRefresh: @escaping () async -> Void,
        onStatusChange: @escaping (String, String) -> Void,
        onDelete: @escaping (String) -> Void,
        onAnswerSent: @escaping () -> Void
    ) {
        self._requests = requests
        self._selectedCategory = selectedCategory
        self._selectedStatus = selectedStatus
        self._searchQuery = searchQuery
        self.onRefresh = onRefresh
        self.onStatusChange = onStatusChange
        self.onDelete = onDelete
        self.onAnswerSent = onAnswerSent
    }

    public var filteredRequests: [EnterpriseRequest] {
        var list = requests
        if selectedCategory != "all" {
            list = list.filter { $0.inquiry_type == selectedCategory }
        }
        if selectedStatus != "all" {
            list = list.filter { $0.status == selectedStatus }
        }
        if !searchQuery.trimmingCharacters(in: .whitespaces).isEmpty {
            let q = searchQuery.lowercased()
            list = list.filter {
                $0.name.lowercased().contains(q) ||
                $0.email.lowercased().contains(q) ||
                ($0.company?.lowercased().contains(q) == true) ||
                ($0.role?.lowercased().contains(q) == true) ||
                ($0.message?.lowercased().contains(q) == true)
            }
        }
        return list
    }

    public var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Top CAD Header Bar
                HStack(spacing: 12) {
                    SwissCADTheme.logoImage
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .frame(width: 30, height: 30)
                        .padding(3)
                        .background(SwissCADTheme.cardDark)
                        .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))

                    VStack(alignment: .leading, spacing: 2) {
                        HStack(spacing: 4) {
                            Rectangle()
                                .fill(SwissCADTheme.emeraldGreen)
                                .frame(width: 6, height: 6)
                            Text("PEGASUS // MOBILE OS")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundColor(SwissCADTheme.electricCyan)
                        }
                        Text("Operator Console")
                            .font(.system(size: 12, weight: .light))
                            .foregroundColor(.white)
                    }

                    Spacer()

                    Button(action: {
                        SwissCADTheme.triggerHaptic()
                        Task { await onRefresh() }
                    }) {
                        Image(systemName: "arrow.clockwise")
                            .font(.system(size: 13, weight: .bold))
                            .foregroundColor(.white)
                            .padding(8)
                            .background(SwissCADTheme.cardDark)
                            .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))
                    }
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
                .background(SwissCADTheme.panelDark)
                .overlay(Rectangle().frame(height: 1).foregroundColor(SwissCADTheme.borderStroke), alignment: .bottom)

                // Category Filter Pills (Horizontal Scroll)
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach([
                            ("all", "All Inquiries", "square.grid.2x2"),
                            ("demo", "Demo Requests", "shippingbox"),
                            ("pilot", "Pilot Programs", "network"),
                            ("general", "General", "envelope")
                        ], id: \.0) { catId, catTitle, icon in
                            let isSel = selectedCategory == catId
                            Button(action: {
                                SwissCADTheme.triggerHaptic()
                                selectedCategory = catId
                            }) {
                                HStack(spacing: 5) {
                                    Image(systemName: icon)
                                        .font(.system(size: 10))
                                    Text(catTitle.uppercased())
                                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                                }
                                .foregroundColor(isSel ? .black : SwissCADTheme.techGray)
                                .padding(.horizontal, 10)
                                .padding(.vertical, 6)
                                .background(isSel ? Color.white : SwissCADTheme.panelDark)
                                .overlay(Rectangle().stroke(isSel ? Color.white : SwissCADTheme.borderStroke, lineWidth: 1))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 8)
                }
                .background(SwissCADTheme.pitchBlack)

                // Status Filter Chips
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 6) {
                        ForEach(["all", "new", "in_review", "contacted", "qualified", "closed"], id: \.self) { st in
                            let isSel = selectedStatus == st
                            let count = st == "all" ? requests.count : requests.filter { $0.status == st }.count
                            let color = st == "all" ? Color.white : SwissCADTheme.statusColor(for: st)

                            Button(action: {
                                SwissCADTheme.triggerHaptic()
                                selectedStatus = st
                            }) {
                                HStack(spacing: 4) {
                                    if st != "all" {
                                        Circle()
                                            .fill(color)
                                            .frame(width: 5, height: 5)
                                    }
                                    Text("\(st.replacingOccurrences(of: "_", with: " ").uppercased()) (\(count))")
                                        .font(.system(size: 9, weight: isSel ? .bold : .medium, design: .monospaced))
                                }
                                .foregroundColor(isSel ? .black : SwissCADTheme.techGray)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 5)
                                .background(isSel ? Color.white : SwissCADTheme.cardDark)
                                .overlay(Rectangle().stroke(isSel ? Color.white : SwissCADTheme.borderStroke, lineWidth: 1))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.bottom, 8)
                }
                .background(SwissCADTheme.pitchBlack)

                // Divider
                Rectangle()
                    .frame(height: 1)
                    .foregroundColor(SwissCADTheme.borderStroke)

                // Queue Content List
                if filteredRequests.isEmpty {
                    VStack(spacing: 16) {
                        Spacer()
                        SwissCADTheme.logoImage
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .frame(width: 60, height: 60)
                            .opacity(0.3)
                        Text("NO CLIENT INQUIRIES FOUND")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundColor(SwissCADTheme.mutedGray)
                        Text("Adjust your category, status, or search keywords.")
                            .font(.system(size: 11))
                            .foregroundColor(SwissCADTheme.techGray)
                        Button("CLEAR ALL FILTERS") {
                            SwissCADTheme.triggerHaptic()
                            selectedCategory = "all"
                            selectedStatus = "all"
                            searchQuery = ""
                        }
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundColor(SwissCADTheme.electricCyan)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(SwissCADTheme.cardDark)
                        .overlay(Rectangle().stroke(SwissCADTheme.borderCyan, lineWidth: 1))
                        Spacer()
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(SwissCADTheme.pitchBlack)
                } else {
                    List {
                        ForEach(filteredRequests) { req in
                            Button(action: {
                                SwissCADTheme.triggerHaptic()
                                activeDetailRequest = req
                            }) {
                                CompactRequestCard(request: req)
                            }
                            .buttonStyle(.plain)
                            .listRowInsets(EdgeInsets(top: 6, leading: 14, bottom: 6, trailing: 14))
                            .listRowBackground(SwissCADTheme.pitchBlack)
                            .listRowSeparator(.hidden)
                            .swipeActions(edge: .leading) {
                                Button {
                                    SwissCADTheme.triggerHaptic()
                                    onStatusChange(req.id, "qualified")
                                } label: {
                                    Label("Qualified", systemImage: "checkmark.seal.fill")
                                }
                                .tint(SwissCADTheme.emeraldGreen)
                            }
                            .swipeActions(edge: .trailing) {
                                Button {
                                    SwissCADTheme.triggerHaptic()
                                    onStatusChange(req.id, "in_review")
                                } label: {
                                    Label("In Review", systemImage: "clock.fill")
                                }
                                .tint(SwissCADTheme.electricCyan)

                                Button {
                                    SwissCADTheme.triggerHaptic()
                                    onStatusChange(req.id, "closed")
                                } label: {
                                    Label("Close", systemImage: "xmark.circle")
                                }
                                .tint(SwissCADTheme.mutedGray)
                            }
                        }
                    }
                    .listStyle(.plain)
                    .background(SwissCADTheme.pitchBlack)
                    .refreshable {
                        await onRefresh()
                    }
                }
            }
            .searchable(text: $searchQuery, prompt: "Search client, company, specs...")
            .background(SwissCADTheme.pitchBlack)
            .sheet(item: $activeDetailRequest) { req in
                RequestDetailSheetHost(
                    request: req,
                    onStatusChange: onStatusChange,
                    onDelete: onDelete,
                    onAnswerSent: onAnswerSent,
                    onDismiss: { activeDetailRequest = nil }
                )
            }
        }
    }
}

// Compact Request Card for Mobile List
private struct CompactRequestCard: View {
    let request: EnterpriseRequest

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(request.name)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundColor(.white)
                    Text(request.email)
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundColor(SwissCADTheme.techGray)
                }

                Spacer()

                // Status Pill
                let col = SwissCADTheme.statusColor(for: request.status)
                HStack(spacing: 4) {
                    Circle()
                        .fill(col)
                        .frame(width: 5, height: 5)
                    Text(request.status.replacingOccurrences(of: "_", with: " ").uppercased())
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundColor(col)
                }
                .padding(.horizontal, 6)
                .padding(.vertical, 3)
                .background(SwissCADTheme.statusBackground(for: request.status))
                .overlay(Rectangle().stroke(col.opacity(0.4), lineWidth: 1))
            }

            if let comp = request.company, !comp.isEmpty {
                Text(comp)
                    .font(.system(size: 12, weight: .medium))
                    .foregroundColor(Color.white.opacity(0.85))
            }

            if let msg = request.message, !msg.isEmpty {
                Text(msg)
                    .font(.system(size: 11))
                    .foregroundColor(SwissCADTheme.techGray)
                    .lineLimit(2)
            }

            HStack {
                Text(request.inquiry_type.uppercased())
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundColor(SwissCADTheme.electricCyan)
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(SwissCADTheme.panelDark)
                    .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))

                Text(request.role ?? "Operations")
                    .font(.system(size: 10))
                    .foregroundColor(SwissCADTheme.mutedGray)

                Spacer()

                if let fleet = request.fleet_size, !fleet.isEmpty {
                    Text(fleet.uppercased())
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundColor(SwissCADTheme.techGray)
                }
            }
        }
        .padding(12)
        .background(SwissCADTheme.cardDark)
        .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))
    }
}

// Mobile Sheet wrapper for detail view
private struct RequestDetailSheetHost: View {
    @State var request: EnterpriseRequest?
    let onStatusChange: (String, String) -> Void
    let onDelete: (String) -> Void
    let onAnswerSent: () -> Void
    let onDismiss: () -> Void

    init(
        request: EnterpriseRequest,
        onStatusChange: @escaping (String, String) -> Void,
        onDelete: @escaping (String) -> Void,
        onAnswerSent: @escaping () -> Void,
        onDismiss: @escaping () -> Void
    ) {
        self._request = State(initialValue: request)
        self.onStatusChange = onStatusChange
        self.onDelete = onDelete
        self.onAnswerSent = onAnswerSent
        self.onDismiss = onDismiss
    }

    var body: some View {
        NavigationStack {
            RequestDetailView(
                request: $request,
                onStatusChange: onStatusChange,
                onDelete: { id in
                    onDelete(id)
                    onDismiss()
                },
                onAnswerSent: onAnswerSent
            )
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("DONE") {
                        onDismiss()
                    }
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundColor(SwissCADTheme.electricCyan)
                }
            }
        }
    }
}
