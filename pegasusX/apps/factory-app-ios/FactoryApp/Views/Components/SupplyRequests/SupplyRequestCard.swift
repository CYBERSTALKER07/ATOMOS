import SwiftUI

struct SupplyRequestCard: View {
    let request: SupplyRequest
    let transitioning: Bool
    let qcResult: String
    let onAction: (String) -> Void
    let onQC: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(supplyRequestLabel(request))
                        .font(.subheadline.bold())
                    Text(L10n.format("mobile_factory.ui.request_prefix", "\(request.id.prefix(8))"))
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                    SupplyTag(text: request.state, emphasized: true)
                    SupplyTag(text: request.priority.isEmpty ? "NORMAL" : request.priority, emphasized: false)
                    if slaBadgeVisible(request.slaStatus) {
                        SupplyTag(
                            text: (request.slaStatus ?? "").replacingOccurrences(of: "_", with: " "),
                            emphasized: request.slaStatus == "BREACHED"
                        )
                    }
                }
            }
            .accessibilityElement(children: .combine)

            HStack(spacing: LabTheme.spacingSM) {
                SupplyMetric(label: "Volume", value: supplyVolumeLabel(request.totalVolumeVU))
                SupplyMetric(label: "Created", value: supplyShortDate(request.createdAt))
                SupplyMetric(label: "Delivery", value: supplyDeliveryWithSLA(request))
            }

            if !request.notes.isEmpty {
                Text(request.notes)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(LabTheme.spacingMD)
                    .background(
                        LabTheme.secondaryBackground,
                        in: RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous)
                    )
            }

            if !qcResult.isEmpty {
                Text("QC \(qcResult)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            HStack(spacing: LabTheme.spacingSM) {
                Button {
                    onQC("PASS")
                } label: {
                    Text("PASS")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .disabled(transitioning)
                .accessibilityLabel("Mark QC pass")

                Button(role: .destructive) {
                    onQC("FAIL")
                } label: {
                    Text("FAIL")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .disabled(transitioning)
                .accessibilityLabel("Mark QC fail")
            }

            let actions = supplyRequestActions(for: request.state)
            if actions.isEmpty {
                Text("mobile_factory.ui.no_manual_action_is_available_for_the_current_state")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            } else {
                HStack(spacing: LabTheme.spacingSM) {
                    ForEach(actions, id: \.action) { action in
                        SupplyActionButton(action: action, transitioning: transitioning) {
                            onAction(action.action)
                        }
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}
