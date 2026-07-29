import SwiftUI
import PhotosUI

struct DeliveryCorrectionSummaryBar: View {
    let vm: CorrectionViewModel
    @Binding var pickedPhoto: PhotosPickerItem?
    @Binding var showConfirmAlert: Bool
    
    var body: some View {
        VStack(spacing: 10) {
            if vm.hasRejections && vm.needsPhotoProof {
                PhotosPicker(selection: $pickedPhoto, matching: .images) {
                    Label(
                        vm.evidencePhotoURL.isEmpty ? "Add damage photo" : "Photo ready — change",
                        systemImage: "camera.fill"
                    )
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(LabTheme.fg.opacity(0.06), in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                if vm.isUploadingPhoto {
                    ProgressView("Uploading…").tint(LabTheme.fg)
                }
                if let img = vm.previewImage {
                    Image(uiImage: img)
                        .resizable()
                        .scaledToFit()
                        .frame(maxHeight: 100)
                        .clipShape(.rect(cornerRadius: 8))
                }
            }
            if let err = vm.submitError {
                Text(err)
                    .font(.caption)
                    .foregroundStyle(LabTheme.destructive)
            }

            // Original total
            HStack {
                Text("Original total")
                    .font(.subheadline)
                    .foregroundStyle(LabTheme.fgSecondary)
                Spacer()
                Text(vm.originalTotal.formattedAmount)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LabTheme.fg)
            }

            // Refund delta
            if vm.refundDelta > 0 {
                HStack {
                    Text("Refund delta")
                        .font(.subheadline)
                        .foregroundStyle(LabTheme.destructive)
                    Spacer()
                    Text("−\(vm.refundDelta.formattedAmount)")
                        .font(.subheadline.weight(.bold))
                        .foregroundStyle(LabTheme.destructive)
                }
            }

            Rectangle()
                .fill(LabTheme.separator)
                .frame(height: 0.5)

            // Adjusted total
            HStack {
                Text("Adjusted total")
                    .font(.headline)
                    .foregroundStyle(LabTheme.fg)
                Spacer()
                Text(vm.adjustedTotal.formattedAmount)
                    .font(.headline)
                    .foregroundStyle(LabTheme.fg)
            }

            // Submit button
            Button {
                if vm.hasRejections { showConfirmAlert = true }
            } label: {
                Text(vm.hasRejections
                     ? "Submit Amendment (\(vm.rejectedCount) rejected)"
                     : "All Items Delivered")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(vm.hasRejections ? .white : LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(
                        vm.hasRejections ? LabTheme.destructive : LabTheme.fg.opacity(0.06),
                        in: .rect(cornerRadius: LabTheme.buttonRadius)
                    )
            }
            .buttonStyle(.pressable)
            .disabled(!vm.hasRejections || vm.isUploadingPhoto || vm.isSubmitting)
        }
        .padding(LabTheme.s16)
        .background(.ultraThinMaterial)
    }
}
