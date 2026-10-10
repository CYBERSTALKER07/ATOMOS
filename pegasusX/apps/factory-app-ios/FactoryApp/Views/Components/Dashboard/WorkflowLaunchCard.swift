import SwiftUI

struct WorkflowLaunchCard: View {
    let onOpenSupplyRequests: () -> Void
    let onOpenPayloadOverride: () -> Void
    let onOpenManifestExceptions: () -> Void
    let onOpenManifests: () -> Void
    let onOpenAnalytics: () -> Void
    let onOpenCreateTransfer: () -> Void
    let onOpenInsights: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            Label {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text("mobile_factory.ui.operator_workflows")
                        .font(.headline)
                    Text("mobile_factory.ui.warehouse_demand_and_live_manifest_overrides_are_available_in_na")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            } icon: {
                Image(systemName: "iphone.gen3")
                    .font(.title3)
                    .foregroundStyle(.secondary)
            }

            WorkflowLaunchRow(
                title: "Supply requests",
                supporting: "Review warehouse demand and advance requests through production states.",
                actionLabel: "Open requests",
                onTap: onOpenSupplyRequests
            )
            WorkflowLaunchRow(
                title: "Payload override",
                supporting: "Move transfers between loading manifests or release them back to approved stock.",
                actionLabel: "Open override",
                onTap: onOpenPayloadOverride
            )
            WorkflowLaunchRow(
                title: "Manifest lifecycle",
                supporting: "Advance manifests through draft, loading, sealed, dispatched, and completed.",
                actionLabel: "Open manifests",
                onTap: onOpenManifests
            )
            WorkflowLaunchRow(
                title: "Gate exceptions",
                supporting: "Review transfers removed from manifests and DLQ escalations.",
                actionLabel: "Open exceptions",
                onTap: onOpenManifestExceptions
            )
            WorkflowLaunchRow(
                title: "Create transfer",
                supporting: "Stage a new factory-to-warehouse movement with volume and optional fleet assignment.",
                actionLabel: "Create transfer",
                onTap: onOpenCreateTransfer
            )
            WorkflowLaunchRow(
                title: "Replenishment insights",
                supporting: "Warehouse stock velocity and reorder pressure linked to this factory.",
                actionLabel: "Open insights",
                onTap: onOpenInsights
            )
            WorkflowLaunchRow(
                title: "Analytics overview",
                supporting: "Factory throughput, active manifests, exception queue, and lead time.",
                actionLabel: "Open analytics",
                onTap: onOpenAnalytics
            )
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .padding(.horizontal)
    }
}

struct WorkflowLaunchRow: View {
    let title: String
    let supporting: String
    let actionLabel: String
    let onTap: () -> Void

    var body: some View {
        HStack(alignment: .center, spacing: LabTheme.spacingMD) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(title)
                    .font(.subheadline.bold())
                Text(supporting)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Button(actionLabel, action: onTap)
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}
