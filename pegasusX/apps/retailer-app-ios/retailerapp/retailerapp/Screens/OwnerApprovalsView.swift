import SwiftUI

struct OwnerApprovalsView: View {
    @State private var vm = OwnerGovernanceViewModel()
    @State private var reasonDrafts: [String: String] = [:]

    var body: some View {
        List {
            if vm.isLoadingRole {
                Section {
                    HStack {
                        Spacer()
                        ProgressView()
                        Spacer()
                    }
                }
            } else if !vm.isOwner {
                Section {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Owner access required")
                            .font(.headline)
                        Text("Only the store owner can approve or reject mandate-exceeding orders.")
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    .padding(.vertical, 4)
                }
            } else {
                Section {
                    Picker("Status", selection: Binding(
                        get: { vm.statusFilter },
                        set: { next in Task { await vm.selectStatusFilter(next) } }
                    )) {
                        Text("Pending").tag("PENDING_OWNER")
                        Text("Approved").tag("APPROVED")
                        Text("Rejected").tag("REJECTED")
                    }
                    .pickerStyle(.segmented)
                }

                if let banner = vm.actionBanner {
                    Section {
                        Text(banner)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(AppTheme.accent)
                    }
                }

                if let error = vm.approvalsError {
                    Section {
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(AppTheme.destructive)
                        Button("common.action.retry") {
                            Task { await vm.refreshApprovals() }
                        }
                    }
                }

                if vm.approvalsLoading && vm.approvals.isEmpty {
                    Section {
                        HStack {
                            Spacer()
                            ProgressView()
                            Spacer()
                        }
                    }
                } else if vm.approvals.isEmpty {
                    Section {
                        Text(String(localized: "mobile_retailer.approval.empty"))
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                } else {
                    Section {
                        ForEach(vm.approvals) { req in
                            approvalRow(req)
                        }
                    }
                }
            }
        }
        .navigationTitle(String(localized: "mobile_retailer.approval.inbox_title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.loadRoleAndApprovals() }
        .refreshable { await vm.refreshApprovals() }
    }

    @ViewBuilder
    private func approvalRow(_ req: RetailerApprovalItem) -> some View {
        let isBusy = vm.decidingRequestId == req.requestId
        let isPending = req.status.uppercased() == "PENDING_OWNER"
        let displayRule = req.trippedRule.replacingOccurrences(of: "_", with: " ").lowercased().capitalized

        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(req.requestedByName.isEmpty ? (req.requestedByUser.isEmpty ? "Seller" : req.requestedByUser) : req.requestedByName)
                    .font(.headline)
                Spacer()
                Text(RetailerMoney.formatWholeSom(req.amountMinor))
                    .font(.headline)
                    .foregroundStyle(AppTheme.accent)
            }

            Text(displayRule)
                .font(.caption)
                .foregroundStyle(AppTheme.textSecondary)

            let meta = [
                req.paymentMethod.isEmpty ? nil : "Method: \(req.paymentMethod)",
                req.payloadKind.isEmpty ? nil : req.payloadKind,
                req.status,
            ].compactMap { $0 }.joined(separator: " · ")
            Text(meta)
                .font(.caption2)
                .foregroundStyle(AppTheme.textTertiary)

            if let note = req.decisionReason, !note.isEmpty {
                Text("Note: \(note)")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
            }

            if isPending {
                TextField("Decision note (optional)", text: Binding(
                    get: { reasonDrafts[req.requestId] ?? "" },
                    set: { reasonDrafts[req.requestId] = $0 }
                ))
                .textFieldStyle(.roundedBorder)
                .disabled(isBusy)

                HStack(spacing: 10) {
                    Button {
                        Task { await vm.approveRequest(req.requestId, reason: reasonDrafts[req.requestId]) }
                    } label: {
                        Text(isBusy ? "…" : String(localized: "mobile_retailer.approval.approve"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(isBusy)

                    Button(role: .destructive) {
                        Task { await vm.rejectRequest(req.requestId, reason: reasonDrafts[req.requestId]) }
                    } label: {
                        Text(String(localized: "mobile_retailer.approval.reject"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .disabled(isBusy)
                }
            }
        }
        .padding(.vertical, 4)
    }
}

struct SellerMandateEditorSheet: View {
    @Bindable var vm: OwnerGovernanceViewModel

    var body: some View {
        NavigationStack {
            Form {
                if let form = vm.mandateForm {
                    Section {
                        Text(form.memberRole.isEmpty ? form.memberName : "\(form.memberName) (\(form.memberRole))")
                            .font(.headline)
                    }

                    if vm.mandateLoading {
                        Section {
                            HStack {
                                Spacer()
                                ProgressView()
                                Spacer()
                            }
                        }
                    } else {
                        Section("Permissions") {
                            Toggle("Can receive goods", isOn: mandateBinding(\.canReceiveGoods))
                            Toggle("Can pay cash", isOn: mandateBinding(\.canPayCash))
                            Toggle("Can use credit", isOn: mandateBinding(\.canUseCredit))
                        }

                        Section("Caps (whole so'm, 0 = unlimited)") {
                            capField(
                                label: String(localized: "mobile_retailer.mandate.daily_credit"),
                                binding: mandateBinding(\.dailyCreditCapSom)
                            )
                            capField(
                                label: String(localized: "mobile_retailer.mandate.daily_cash"),
                                binding: mandateBinding(\.dailyCashCapSom)
                            )
                            capField(
                                label: String(localized: "mobile_retailer.mandate.max_order"),
                                binding: mandateBinding(\.maxSingleOrderSom)
                            )
                        }

                        Section("Today usage") {
                            Text("Credit: \(RetailerMoney.formatWholeSom(form.dailyUsage.spentCreditMinor))")
                            Text("Cash: \(RetailerMoney.formatWholeSom(form.dailyUsage.spentCashMinor))")
                            Text("Orders: \(form.dailyUsage.orderCount)")
                        }
                    }

                    if let error = vm.mandateError, !error.isEmpty {
                        Section {
                            Text(error)
                                .font(.caption)
                                .foregroundStyle(AppTheme.destructive)
                        }
                    }
                }
            }
            .navigationTitle(String(localized: "mobile_retailer.mandate.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") {
                        vm.mandateForm = nil
                    }
                    .disabled(vm.mandateSaving)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(vm.mandateSaving ? "…" : String(localized: "mobile_retailer.mandate.save")) {
                        Task { await vm.saveMandate() }
                    }
                    .disabled(vm.mandateLoading || vm.mandateSaving || vm.mandateForm == nil)
                }
            }
        }
    }

    private func capField(label: String, binding: Binding<String>) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption)
                .foregroundStyle(AppTheme.textSecondary)
            TextField("0", text: binding)
                .keyboardType(.numberPad)
        }
    }

    private func mandateBinding<T>(_ keyPath: WritableKeyPath<SellerMandateFormState, T>) -> Binding<T> {
        Binding(
            get: {
                guard let form = vm.mandateForm else {
                    return SellerMandateFormState(userId: "", memberName: "")[keyPath: keyPath]
                }
                return form[keyPath: keyPath]
            },
            set: { value in
                vm.mandateForm?[keyPath: keyPath] = value
                vm.mandateError = nil
            }
        )
    }
}
