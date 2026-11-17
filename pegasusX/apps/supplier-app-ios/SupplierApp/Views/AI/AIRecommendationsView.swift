import SwiftUI

struct AIRecommendationsView: View {
    private let statusFilters = ["PENDING", "ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "ALL"]
    private let decisions = ["ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "REOPENED"]

    @State private var viewModel = AIRecommendationsViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.items.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.recommendations")
            } else if let error = viewModel.error, viewModel.items.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else {
                ResponsiveGridContentWrapper {
                    Section {
                        Picker("mobile_supplier.chrome.status", selection: $vm.filter) {
                            ForEach(statusFilters, id: \.self) { Text($0).tag($0) }
                        }
                        .pickerStyle(.segmented)
                        if let feedback = viewModel.feedback {
                            Text(feedback).font(.footnote).foregroundStyle(.secondary)
                        }
                    }

                    if viewModel.items.isEmpty {
                        Section {
                            Text(L10n.format("mobile_supplier.ui.no_lowercased_advisory_rows_for_this_supplier", "\(viewModel.filter.lowercased())"))
                                .foregroundStyle(.secondary)
                        }
                    }

                    ForEach(viewModel.items) { rec in
                        Section {
                            recommendationCard(rec)
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("mobile_supplier.ui.ai_recommendations")
        .task(id: viewModel.filter) { await viewModel.load() }
    }

    @ViewBuilder
    private func recommendationCard(_ rec: SupplierAIRecommendation) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(rec.action).font(.headline)
                Spacer()
                Text(rec.status).font(.caption).foregroundStyle(.secondary)
            }
            Text(L10n.format("mobile_supplier.chrome.confidence_score", String(format: "%.0f", rec.confidence * 100), String(format: "%.2f", rec.score)))
                .font(.caption)
                .foregroundStyle(.secondary)
            if !rec.explanation.isEmpty {
                Text(rec.explanation).font(.subheadline)
            }
            Text(L10n.format("mobile_supplier.ui.aggregatetype_aggregateid_source_source", "\(rec.aggregateType)", "\(rec.aggregateId)", "\(rec.source)"))
                .font(.caption2)
                .foregroundStyle(.secondary)

            if !rec.reasonCodes.isEmpty {
                Text(L10n.format("mobile_supplier.ui.reasons_joined", "\(rec.reasonCodes.joined(separator: ", "))"))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            ForEach(rec.evidence) { ev in
                HStack {
                    Text(ev.label).foregroundStyle(.secondary)
                    Spacer()
                    Text(ev.value)
                }
                .font(.caption)
            }

            if let decision = rec.decision, !decision.isEmpty {
                Text(L10n.format("mobile_supplier.chrome.decision_by", decision, rec.decidedBy ?? L10n.format("mobile_supplier.chrome.operator")))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if rec.decision == nil || rec.decision?.isEmpty == true {
                HStack {
                    ForEach(decisions, id: \.self) { decision in
                        Button(decision.capitalized) {
                            Task { await viewModel.record(rec, decision: decision) }
                        }
                        .font(.caption)
                        .buttonStyle(.bordered)
                        .frame(minHeight: 44)
                        .disabled(viewModel.pendingDecisionId != nil)
                    }
                }
            }
        }
    }
}
