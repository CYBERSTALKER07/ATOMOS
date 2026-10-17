import SwiftUI

public struct RequestListView: View {
    public let requests: [EnterpriseRequest]
    @Binding public var selectedRequest: EnterpriseRequest?
    @Binding public var searchQuery: String

    public var filteredRequests: [EnterpriseRequest] {
        if searchQuery.isEmpty { return requests }
        let query = searchQuery.lowercased()
        return requests.filter {
            $0.name.lowercased().contains(query) ||
            $0.email.lowercased().contains(query) ||
            ($0.company?.lowercased().contains(query) ?? false) ||
            ($0.role?.lowercased().contains(query) ?? false)
        }
    }

    public var body: some View {
        VStack(spacing: 0) {
            // Header Bar
            HStack {
                Text("\(filteredRequests.count) QUEUED REQUESTS")
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundColor(SwissCADTheme.techGray)
                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(SwissCADTheme.panelDark)
            .overlay(Rectangle().frame(height: 1).foregroundColor(SwissCADTheme.borderStroke), alignment: .bottom)

            if filteredRequests.isEmpty {
                VStack(spacing: 8) {
                    Spacer()
                    Image(systemName: "tray")
                        .font(.system(size: 32))
                        .foregroundColor(SwissCADTheme.mutedGray)
                    Text("NO REQUESTS FOUND")
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundColor(SwissCADTheme.mutedGray)
                    Spacer()
                }
            } else {
                List(filteredRequests, selection: $selectedRequest) { req in
                    VStack(alignment: .leading, spacing: 6) {
                        HStack {
                            Text(req.name)
                                .font(.system(size: 13, weight: .semibold))
                                .foregroundColor(.white)

                            Spacer()

                            Text(req.status.replacingOccurrences(of: "_", with: " ").uppercased())
                                .font(.system(size: 9, weight: .bold, design: .monospaced))
                                .foregroundColor(SwissCADTheme.statusColor(for: req.status))
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(SwissCADTheme.statusBackground(for: req.status))
                                .overlay(
                                    Rectangle()
                                        .stroke(SwissCADTheme.statusColor(for: req.status), lineWidth: 1)
                                )
                        }

                        if let company = req.company, !company.isEmpty {
                            Text(company)
                                .font(.system(size: 11))
                                .foregroundColor(SwissCADTheme.techGray)
                        }

                        HStack {
                            if let role = req.role {
                                Text(role)
                                    .font(.system(size: 10, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.electricCyan)
                            }
                            Spacer()
                            Text(req.displayDate)
                                .font(.system(size: 9, design: .monospaced))
                                .foregroundColor(SwissCADTheme.mutedGray)
                        }
                    }
                    .padding(.vertical, 8)
                    .padding(.horizontal, 10)
                    .cadCard(isSelected: selectedRequest?.id == req.id)
                    .contentShape(Rectangle())
                    .onTapGesture {
                        selectedRequest = req
                    }
                    .listRowInsets(EdgeInsets(top: 4, leading: 10, bottom: 4, trailing: 10))
                    .listRowBackground(Color.clear)
                }
                .listStyle(.plain)
                .background(SwissCADTheme.pitchBlack)
            }
        }
    }
}
