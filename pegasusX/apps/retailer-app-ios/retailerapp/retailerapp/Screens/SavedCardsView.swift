import SwiftUI

struct SavedCardItem: Identifiable, Codable {
    var id: String
    var panMasked: String
    var scheme: String // UZCARD, HUMO, VISA, MASTERCARD
    var holder: String
    var expiry: String
    var bankName: String
    var isDefault: Bool
}

struct SavedCardsView: View {
    @State private var vm = SavedCardsViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            Section("Saqlangan Kartalar") {
                if vm.cards.isEmpty {
                    Text("Hozircha saqlangan kartalar mavjud emas.")
                        .foregroundStyle(AppTheme.textSecondary)
                } else {
                    ForEach(vm.cards) { card in
                        HStack(spacing: 12) {
                            schemeIcon(card.scheme)
                            VStack(alignment: .leading, spacing: 3) {
                                HStack {
                                    Text(card.panMasked)
                                        .font(.system(.body, design: .monospaced))
                                        .fontWeight(.semibold)
                                    if card.isDefault {
                                        Text("Asosiy")
                                            .font(.caption2)
                                            .fontWeight(.bold)
                                            .padding(.horizontal, 6)
                                            .padding(.vertical, 2)
                                            .background(AppTheme.successSoft)
                                            .foregroundStyle(AppTheme.success)
                                            .clipShape(Capsule())
                                    }
                                }
                                Text("\(card.bankName) · \(card.expiry)")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            Spacer()
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(card.scheme) \(card.panMasked), \(card.bankName)\(card.isDefault ? ", asosiy karta" : "")")
                        .swipeActions {
                            Button(role: .destructive) {
                                vm.deleteCard(card.id)
                            } label: {
                                Label("O'chirish", systemImage: "trash")
                            }
                            if !card.isDefault {
                                Button {
                                    vm.setDefaultCard(card.id)
                                } label: {
                                    Label("Asosiy qilish", systemImage: "star")
                                }
                                .tint(.orange)
                            }
                        }
                    }
                }
            }

            Section {
                Button {
                    Haptics.light()
                    vm.showingAddCard = true
                } label: {
                    Label("Yangi Karta Qo'shish (GlobalPay)", systemImage: "creditcard.badge.plus")
                        .fontWeight(.semibold)
                        .foregroundStyle(AppTheme.accent)
                        .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("Yangi karta qo'shish")
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("To'lov Kartalari")
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $vm.showingAddCard) {
            NavigationStack {
                Form {
                    if vm.bindingStep == 0 {
                        Section("Karta Ma'lumotlari") {
                            TextField("Karta raqami (16 ta raqam)", text: $vm.panInput)
                                .keyboardType(.numberPad)
                                .accessibilityLabel("Karta raqami")
                            TextField("Amal qilish muddati (MM/YY)", text: $vm.expInput)
                                .keyboardType(.numbersAndPunctuation)
                                .accessibilityLabel("Amal qilish muddati")
                        }
                        Section {
                            Button("SMS Tasdiqlash Kodini Olish") {
                                vm.requestOTP()
                            }
                            .pegasusMinimumHitTarget()
                            .disabled(vm.panInput.count < 16)
                            .accessibilityLabel("SMS tasdiqlash kodini olish")
                        }
                    } else {
                        Section("SMS Tasdiqlash Kodingiz") {
                            if let statusMessage = vm.statusMessage {
                                Text(statusMessage)
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            TextField("6 xonali SMS kod (123456)", text: $vm.otpInput)
                                .keyboardType(.numberPad)
                                .accessibilityLabel("SMS kod")
                            Button("Avtoto'ldirish (123456)") {
                                vm.autofillOTP()
                            }
                            .font(.caption)
                            .pegasusMinimumHitTarget()
                            .accessibilityLabel("Avtoto'ldirish test kodi")
                        }
                        Section {
                            Button("Kartani Bog'lash") {
                                vm.completeCardBinding()
                            }
                            .pegasusMinimumHitTarget()
                            .disabled(vm.otpInput.count < 6)
                            .accessibilityLabel("Kartani bog'lash")
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("Karta Bog'lash")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Bekor qilish") {
                            vm.resetAddCardState()
                            vm.showingAddCard = false
                        }
                        .pegasusMinimumHitTarget()
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func schemeIcon(_ scheme: String) -> some View {
        ZStack {
            RoundedRectangle(cornerRadius: 6)
                .fill(schemeColor(scheme))
                .frame(width: 36, height: 26)
            Text(scheme.prefix(2))
                .font(.system(.caption2, design: .rounded, weight: .bold))
                .foregroundStyle(.white)
        }
    }

    private func schemeColor(_ scheme: String) -> Color {
        switch scheme.uppercased() {
        case "UZCARD": return Color.blue
        case "HUMO": return Color.orange
        case "VISA": return Color(red: 0.1, green: 0.15, blue: 0.45)
        case "MASTERCARD": return Color.red
        default: return Color.gray
        }
    }
}
