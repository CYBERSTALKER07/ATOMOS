import SwiftUI

public struct ContentView: View {
    #if os(iOS)
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    #endif

    @StateObject private var apiClient = EnterpriseAPIClient.shared

    @State private var requests: [EnterpriseRequest] = []
    @State private var selectedRequest: EnterpriseRequest?
    @State private var selectedCategory: String = "all"
    @State private var selectedStatus: String = "all"
    @State private var searchQuery: String = ""

    public init() {}

    private var isCompact: Bool {
        #if os(iOS)
        return horizontalSizeClass == .compact
        #else
        return false
        #endif
    }

    public var body: some View {
        Group {
            if isCompact {
                // iPhone / Compact Width Experience
                CompactQueueView(
                    requests: $requests,
                    selectedCategory: $selectedCategory,
                    selectedStatus: $selectedStatus,
                    searchQuery: $searchQuery,
                    onRefresh: { await refresh() },
                    onStatusChange: { id, newStatus in
                        Task {
                            try? await apiClient.updateStatus(id: id, status: newStatus)
                            await refresh()
                        }
                    },
                    onDelete: { id in
                        Task {
                            try? await apiClient.deleteRequest(id: id)
                            await refresh()
                        }
                    },
                    onAnswerSent: {
                        Task { await refresh() }
                    }
                )
            } else {
                // iPad / Mac / Regular Width 3-Column Experience
                NavigationSplitView {
                    SidebarView(
                        selectedCategory: $selectedCategory,
                        selectedStatus: $selectedStatus,
                        requests: requests
                    )
                    .navigationSplitViewColumnWidth(min: 200, ideal: 240, max: 300)
                } content: {
                    RequestListView(
                        requests: filteredList,
                        selectedRequest: $selectedRequest,
                        searchQuery: $searchQuery
                    )
                    .navigationSplitViewColumnWidth(min: 280, ideal: 340, max: 450)
                } detail: {
                    RequestDetailView(
                        request: $selectedRequest,
                        onStatusChange: { id, newStatus in
                            Task {
                                try? await apiClient.updateStatus(id: id, status: newStatus)
                                await refresh()
                            }
                        },
                        onDelete: { id in
                            Task {
                                try? await apiClient.deleteRequest(id: id)
                                selectedRequest = nil
                                await refresh()
                            }
                        },
                        onAnswerSent: {
                            Task {
                                await refresh()
                            }
                        }
                    )
                }
                .navigationSplitViewStyle(.balanced)
                .toolbar {
                    ToolbarItem(placement: .automatic) {
                        HStack(spacing: 8) {
                            Circle()
                                .fill(apiClient.isOnline ? SwissCADTheme.emeraldGreen : SwissCADTheme.laserOrange)
                                .frame(width: 8, height: 8)
                            Text(apiClient.isOnline ? "SUPABASE LIVE" : "OFFLINE CACHE")
                                .font(.system(size: 9, weight: .bold, design: .monospaced))
                                .foregroundColor(SwissCADTheme.techGray)
                        }
                    }

                    ToolbarItem(placement: .automatic) {
                        Button(action: {
                            SwissCADTheme.triggerHaptic()
                            Task { await refresh() }
                        }) {
                            HStack(spacing: 4) {
                                if apiClient.isSyncing {
                                    ProgressView().controlSize(.small)
                                } else {
                                    Image(systemName: "arrow.clockwise")
                                }
                                Text("REFRESH")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                            }
                        }
                        .keyboardShortcut("r", modifiers: .command)
                    }
                }
            }
        }
        .task {
            await refresh()
        }
        .onChange(of: selectedCategory) { Task { await refresh() } }
        .onChange(of: selectedStatus) { Task { await refresh() } }
    }

    private var filteredList: [EnterpriseRequest] {
        var list = requests
        if selectedCategory != "all" {
            list = list.filter { $0.inquiry_type == selectedCategory }
        }
        if selectedStatus != "all" {
            list = list.filter { $0.status == selectedStatus }
        }
        return list
    }

    private func refresh() async {
        let fetched = await apiClient.fetchRequests(status: selectedStatus, inquiryType: selectedCategory)
        self.requests = fetched
        if selectedRequest == nil, let first = fetched.first {
            selectedRequest = first
        }
    }
}
