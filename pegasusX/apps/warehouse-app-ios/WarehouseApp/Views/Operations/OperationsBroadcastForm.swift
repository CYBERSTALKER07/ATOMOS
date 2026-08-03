import SwiftUI

struct OperationsBroadcastForm: View {
    let broadcastRoles: [String]
    let templates: [BroadcastTemplate]
    
    @Binding var templateDate: String
    @Binding var customReason: String
    @Binding var title: String
    @Binding var broadcastRole: String
    @Binding var bodyText: String
    @Binding var saveAsTemplate: Bool
    
    let broadcasting: Bool
    let savingTemplate: Bool
    
    let onApplyTemplate: (BroadcastTemplate) -> Void
    let onDeleteTemplate: (BroadcastTemplate) async -> Void
    let onSendBroadcast: () async -> Void
    
    var body: some View {
        Group {
            Section {
                WarehouseSectionHeader(
                    title: "Broadcast templates",
                    subtitle: "Built-in depot starters plus your saved custom messages."
                )
            }
            Section {
                if templates.isEmpty {
                    Text("No templates available.")
                        .foregroundStyle(.secondary)
                } else {
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: LabTheme.spacingSM) {
                            ForEach(templates) { template in
                                HStack(spacing: 4) {
                                    Button {
                                        onApplyTemplate(template)
                                    } label: {
                                        let suffix = template.source == "custom" ? " · saved" : ""
                                        Text(template.title + suffix)
                                    }
                                    .buttonStyle(.bordered)
                                    .font(.caption)

                                    if template.source == "custom" {
                                        Button(role: .destructive) {
                                            Task { await onDeleteTemplate(template) }
                                        } label: {
                                            Image(systemName: "xmark.circle.fill")
                                        }
                                        .buttonStyle(.borderless)
                                        .font(.caption)
                                    }
                                }
                            }
                        }
                    }
                }
            }

            Section {
                WarehouseSectionHeader(title: "Send depot broadcast")
            }
            Section {
                TextField("Effective date (optional)", text: $templateDate)
                    .textInputAutocapitalization(.never)
                TextField("Custom reason (optional)", text: $customReason)
                TextField("Title", text: $title)
                Picker("Target role", selection: $broadcastRole) {
                    ForEach(broadcastRoles, id: \.self) { role in
                        Text(role).tag(role)
                    }
                }
                TextField("Message", text: $bodyText, axis: .vertical)
                    .lineLimit(4...8)
                Toggle("Save as custom template for this depot", isOn: $saveAsTemplate)
                Button(broadcasting || savingTemplate ? "Sending…" : "Send broadcast") {
                    Task { await onSendBroadcast() }
                }
                .disabled(
                    broadcasting || savingTemplate
                        || title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        || bodyText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                )
            }
        }
    }
}
