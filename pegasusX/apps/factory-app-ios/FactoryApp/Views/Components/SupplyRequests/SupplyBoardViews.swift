import SwiftUI

struct SupplyBoard: View {
    let requests: [SupplyRequest]
    let transitioningID: String?
    let qcById: [String: String]
    let onAction: (SupplyRequest, String) -> Void
    let onQC: (SupplyRequest, String) -> Void

    private let lanes = ["SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY"]

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(alignment: .top, spacing: LabTheme.spacingSM) {
                ForEach(lanes, id: \.self) { lane in
                    VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                        Text(lane.replacingOccurrences(of: "_", with: " "))
                            .font(.footnote.bold())
                        ForEach(requests.filter { $0.state == lane }) { request in
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(supplyRequestLabel(request))
                                    .font(.subheadline.bold())
                                Text(request.priority.isEmpty ? "NORMAL" : request.priority)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if let qc = qcById[request.id], !qc.isEmpty {
                                    Text("QC \(qc)")
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                                ForEach(supplyRequestActions(for: request.state), id: \.action) { action in
                                    SupplyActionButton(action: action, transitioning: transitioningID == request.id) {
                                        onAction(request, action.action)
                                    }
                                }
                                HStack(spacing: LabTheme.spacingSM) {
                                    Button {
                                        onQC(request, "PASS")
                                    } label: {
                                        Text("PASS")
                                            .frame(minWidth: 44, minHeight: 44)
                                            .contentShape(Rectangle())
                                    }
                                    .disabled(transitioningID == request.id)
                                    .accessibilityLabel("Mark QC pass for \(supplyRequestLabel(request))")

                                    Button(role: .destructive) {
                                        onQC(request, "FAIL")
                                    } label: {
                                        Text("FAIL")
                                            .frame(minWidth: 44, minHeight: 44)
                                            .contentShape(Rectangle())
                                    }
                                    .disabled(transitioningID == request.id)
                                    .accessibilityLabel("Mark QC fail for \(supplyRequestLabel(request))")
                                }
                                .font(.caption)
                            }
                            .frame(width: 220, alignment: .leading)
                            .labCard()
                        }
                    }
                }
            }
        }
    }
}

struct SupplyActionButton: View {
    let action: SupplyRequestActionSpec
    let transitioning: Bool
    let onTap: () -> Void

    var body: some View {
        Group {
            if action.emphasized {
                Button(action: onTap) {
                    Text(action.title)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.borderedProminent)
            } else {
                Button(action: onTap) {
                    Text(action.title)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.bordered)
            }
        }
        .disabled(transitioning)
        .frame(maxWidth: .infinity)
        .accessibilityLabel(action.title)
    }
}
