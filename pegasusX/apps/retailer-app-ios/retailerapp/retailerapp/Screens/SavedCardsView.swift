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
    @State private var cards: [SavedCardItem] = [
        SavedCardItem(
            id: "card-1",
            panMasked: "8600 •••• •••• 0001",
            scheme: "UZCARD",
            holder: "ALISHER NAVOIY",
            expiry: "12/28",
            bankName: "O'zsanoatqurilishbank",
            isDefault: true
        ),
        SavedCardItem(
            id: "card-2",
            panMasked: "9860 •••• •••• 0002",
            scheme: "HUMO",
            holder: "ZULFIYA ISROILOVA",
            expiry: "08/27",
            bankName: "Ipak Yo'li Bank",
            isDefault: false
        )
    ]
    @State private var showingAddCard = false
    @State private var panInput = ""
    @State private var expInput = ""
    @State private var otpInput = ""
    @State private var bindingStep = 0 // 0: input, 1: otp
    @State private var statusMessage: String?

    var body: some View {
        List {
            Section("Saqlangan Kartalar") {
                if cards.isEmpty {
                    Text("Hozircha saqlangan kartalar mavjud emas.")
                        .foregroundStyle(AppTheme.textSecondary)
                } else {
                    ForEach(cards) { card in
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
                        .swipeActions {
                            Button(role: .destructive) {
                                deleteCard(card.id)
                            } label: {
                                Label("O'chirish", systemImage: "trash")
                            }
                            if !card.isDefault {
                                Button {
                                    setDefaultCard(card.id)
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
                    showingAddCard = true
                } label: {
                    Label("Yangi Karta Qo'shish (GlobalPay)", systemImage: "creditcard.badge.plus")
                        .fontWeight(.semibold)
                        .foregroundStyle(AppTheme.accent)
                }
            }
        }
        .navigationTitle("To'lov Kartalari")
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $showingAddCard) {
            NavigationStack {
                Form {
                    if bindingStep == 0 {
                        Section("Karta Ma'lumotlari") {
                            TextField("Karta raqami (16 ta raqam)", text: $panInput)
                                .keyboardType(.numberPad)
                            TextField("Amal qilish muddati (MM/YY)", text: $expInput)
                                .keyboardType(.numbersAndPunctuation)
                        }
                        Section {
                            Button("SMS Tasdiqlash Kodini Olish") {
                                bindingStep = 1
                                statusMessage = "SMS kod yuborildi: +998 90 *** ** 01"
                            }
                            .disabled(panInput.count < 16)
                        }
                    } else {
                        Section("SMS Tasdiqlash Kodingiz") {
                            if let statusMessage {
                                Text(statusMessage)
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            TextField("6 xonali SMS kod (123456)", text: $otpInput)
                                .keyboardType(.numberPad)
                            Button("Avtoto'ldirish (123456)") {
                                otpInput = "123456"
                            }
                            .font(.caption)
                        }
                        Section {
                            Button("Kartani Bog'lash") {
                                completeCardBinding()
                            }
                            .disabled(otpInput.count < 6)
                        }
                    }
                }
                .navigationTitle("Karta Bog'lash")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Bekor qilish") {
                            resetAddCardState()
                            showingAddCard = false
                        }
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
                .font(.system(size: 11, weight: .black, design: .rounded))
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

    private func deleteCard(_ id: String) {
        cards.removeAll { $0.id == id }
    }

    private func setDefaultCard(_ id: String) {
        for i in cards.indices {
            cards[i].isDefault = (cards[i].id == id)
        }
    }

    private func completeCardBinding() {
        let clean = panInput.replacingOccurrences(of: " ", with: "")
        let last4 = clean.count >= 4 ? String(clean.suffix(4)) : "0000"
        let scheme = clean.hasPrefix("9860") ? "HUMO" : "UZCARD"
        let newCard = SavedCardItem(
            id: "card-\(UUID().uuidString.prefix(6))",
            panMasked: "\(clean.prefix(4)) •••• •••• \(last4)",
            scheme: scheme,
            holder: "YANGI KARTA",
            expiry: expInput.isEmpty ? "12/28" : expInput,
            bankName: "O'zbekiston Banki",
            isDefault: cards.isEmpty
        )
        cards.append(newCard)
        resetAddCardState()
        showingAddCard = false
    }

    private func resetAddCardState() {
        panInput = ""
        expInput = ""
        otpInput = ""
        bindingStep = 0
        statusMessage = nil
    }
}
