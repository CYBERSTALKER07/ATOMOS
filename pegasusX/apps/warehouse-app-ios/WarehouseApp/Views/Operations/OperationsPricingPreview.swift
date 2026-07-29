import SwiftUI

struct OperationsPricingPreview: View {
    @Binding var productId: String
    @Binding var retailerId: String
    @Binding var proposedPrice: String
    
    let previewLoading: Bool
    let preview: RetailerOverridePreview?
    
    let onSchedulePreview: () -> Void
    
    var body: some View {
        Group {
            Section {
                WarehouseSectionHeader(
                    title: "Pricing impact preview (read-only)",
                    subtitle: "Preview proposed retailer price vs catalog list price. Does not create overrides."
                )
            }
            Section {
                TextField("Product / SKU ID", text: $productId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onChange(of: productId) { _, _ in onSchedulePreview() }
                TextField("Retailer ID (optional)", text: $retailerId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onChange(of: retailerId) { _, _ in onSchedulePreview() }
                TextField("Proposed price (minor units)", text: $proposedPrice)
                    .keyboardType(.numberPad)
                    .onChange(of: proposedPrice) { _, _ in onSchedulePreview() }

                if previewLoading {
                    Text("Loading preview…")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                if let preview {
                    LabeledContent("Retailers on SKU", value: "\(preview.retailersOnSkuCount)")
                    LabeledContent("Active overrides", value: "\(preview.activeOverrideCount)")
                    LabeledContent("Catalog list price", value: "\(preview.catalogListPrice)")
                    LabeledContent("Margin delta / unit", value: "\(preview.marginDeltaPerUnit)")
                    Text(preview.marginEstimateLabel)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                    if preview.readOnly == true {
                        Text("Read-only — contact supplier to apply overrides.")
                            .font(.footnote.weight(.medium))
                    }
                }
            }
        }
    }
}
