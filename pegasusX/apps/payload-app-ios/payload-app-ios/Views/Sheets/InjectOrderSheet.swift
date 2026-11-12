import Observation
import SwiftUI

@Observable
@MainActor
final class InjectOrderViewModel {
    var orderId: String = ""
    var showScanner: Bool = false

    var trimmedOrderId: String {
        orderId.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    var canSubmit: Bool {
        !trimmedOrderId.isEmpty
    }

    func toggleScanner() {
        PegasusHaptics.selection()
        showScanner.toggle()
    }

    func applyScannedBarcode(_ raw: String) {
        orderId = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        showScanner = false
        PegasusHaptics.success()
    }
}

struct InjectOrderSheet: View {
    let injecting: Bool
    let onCancel: () -> Void
    let onSubmit: (String) -> Void
    @State private var form = InjectOrderViewModel()

    var body: some View {
        @Bindable var form = form
        ZStack {
            TermTheme.bg.ignoresSafeArea()

            ScrollView {
                VStack(spacing: 24) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("SYSTEM_INJECTION")
                                .font(.system(.caption, design: .monospaced, weight: .black))
                                .foregroundStyle(TermTheme.secondary)
                            Text("INJECT_ORDER")
                                .font(.system(.title2, design: .monospaced, weight: .black))
                                .foregroundStyle(TermTheme.accent)
                        }
                        Spacer()
                        Button {
                            PegasusHaptics.light()
                            onCancel()
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .font(.system(.title))
                                .frame(minWidth: 44, minHeight: 44)
                                .foregroundStyle(TermTheme.tertiary)
                        }
                        .accessibilityLabel("Cancel injection")
                        .disabled(injecting)
                    }
                    .padding(.horizontal, 4)

                    VStack(alignment: .leading, spacing: 16) {
                        Text("TARGET_ORDER_ID")
                            .font(.system(.caption2, design: .monospaced, weight: .bold))
                            .foregroundStyle(TermTheme.secondary)

                        TextField("mobile_payload.ui.ord_xxxxxx", text: $form.orderId)
                            .font(.system(.title3, design: .monospaced, weight: .black))
                            .padding(16)
                            .frame(minHeight: 44)
                            .background(TermTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                            .overlay {
                                RoundedRectangle(cornerRadius: 12, style: .continuous)
                                    .stroke(TermTheme.accent.opacity(0.1), lineWidth: 1)
                            }
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .disabled(injecting)
                            .accessibilityLabel("Target order identifier")

                        Text("mobile_payload.ui.add_an_order_mid_load_scan_an_order_label_or_enter_the_order_id")
                            .font(.system(.caption, design: .monospaced, weight: .medium))
                            .foregroundStyle(TermTheme.tertiary)
                            .padding(.horizontal, 4)

                        Button {
                            form.toggleScanner()
                        } label: {
                            Label(form.showScanner ? "HIDE SCANNER" : "SCAN ORDER LABEL", systemImage: "barcode.viewfinder")
                                .font(.system(.caption, design: .monospaced, weight: .bold))
                                .frame(maxWidth: .infinity, minHeight: 44)
                        }
                        .buttonStyle(.bordered)
                        .disabled(injecting)

                        if form.showScanner {
                            EANBarcodeScannerView(onBarcode: { scanned in
                                form.applyScannedBarcode(scanned)
                            })
                        }
                    }
                    .padding(20)
                    .background(TermTheme.card.opacity(0.5))
                    .clipShape(RoundedRectangle(cornerRadius: 20, style: .continuous))
                    .tacticalCard()

                    Button {
                        PegasusHaptics.medium()
                        onSubmit(form.trimmedOrderId)
                    } label: {
                        HStack {
                            if injecting {
                                ProgressView().tint(TermTheme.card)
                            } else {
                                Image(systemName: "bolt.fill")
                                Text("EXECUTE_INJECTION")
                                    .font(.system(.callout, design: .monospaced, weight: .black))
                            }
                        }
                        .frame(maxWidth: .infinity, minHeight: 52)
                        .padding(.vertical, 8)
                        .background(!form.canSubmit ? TermTheme.tertiary : TermTheme.accent)
                        .foregroundStyle(TermTheme.card)
                        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                    }
                    .disabled(injecting || !form.canSubmit)
                    .buttonStyle(.tactical)
                    .accessibilityLabel("Execute order injection")
                }
                .padding(24)
            }
            .scrollDismissesKeyboard(.interactively)
        }
    }
}
