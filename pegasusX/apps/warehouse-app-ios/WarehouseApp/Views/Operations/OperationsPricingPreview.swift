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
                TextField("warehouse_portal.residual.text.product_sku_id", text: $productId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onChange(of: productId) { _, _ in onSchedulePreview() }
                TextField("warehouse_portal.residual.text.retailer_id_optional", text: $retailerId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onChange(of: retailerId) { _, _ in onSchedulePreview() }
                TextField("warehouse_portal.residual.text.proposed_price_minor_units", text: $proposedPrice)
                    .keyboardType(.numberPad)
                    .onChange(of: proposedPrice) { _, _ in onSchedulePreview() }

                if previewLoading {
                    Text("warehouse_portal.operations.operations_pricing_preview.text.loading_preview")
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
                        Text("warehouse_portal.operations.operations_pricing_preview.text.read_only_contact_supplier_to_apply_overrides")
                            .font(.footnote.weight(.medium))
                    }
                }
            }
        }
    }
}
