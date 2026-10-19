import SwiftUI

public struct RequestDetailView: View {
    @Binding public var request: EnterpriseRequest?
    public let onStatusChange: (String, String) -> Void
    public let onDelete: (String) -> Void
    public let onAnswerSent: () -> Void

    // Reply Composer State
    @State private var replySubject: String = ""
    @State private var replyMessage: String = ""
    @State private var internalNotes: String = ""
    @State private var isSendingReply: Bool = false
    @State private var replySuccessNotice: String? = nil
    @State private var replyErrorNotice: String? = nil

    public init(
        request: Binding<EnterpriseRequest?>,
        onStatusChange: @escaping (String, String) -> Void,
        onDelete: @escaping (String) -> Void,
        onAnswerSent: @escaping () -> Void
    ) {
        self._request = request
        self.onStatusChange = onStatusChange
        self.onDelete = onDelete
        self.onAnswerSent = onAnswerSent
    }

    public var body: some View {
        Group {
            if let req = request {
                ScrollView {
                    VStack(alignment: .leading, spacing: 20) {
                        // 1. Header Inspector
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                Text("PEGASUS RECORD // \(req.id.prefix(8).uppercased())")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.electricCyan)

                                Spacer()

                                Text(req.status.replacingOccurrences(of: "_", with: " ").uppercased())
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.statusColor(for: req.status))
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 3)
                                    .background(SwissCADTheme.statusBackground(for: req.status))
                                    .overlay(Rectangle().stroke(SwissCADTheme.statusColor(for: req.status), lineWidth: 1))
                            }

                            Text(req.name)
                                .font(.system(size: 24, weight: .light))
                                .foregroundColor(.white)

                            if let company = req.company {
                                Text(company)
                                    .font(.system(size: 14))
                                    .foregroundColor(SwissCADTheme.techGray)
                            }
                        }
                        .padding(.bottom, 12)
                        .overlay(Rectangle().frame(height: 1).foregroundColor(SwissCADTheme.borderStroke), alignment: .bottom)

                        // 2. Technical Grid Specs (Client Review)
                        VStack(alignment: .leading, spacing: 12) {
                            Text("CLIENT SPECIFICATIONS & TELEMETRY")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundColor(SwissCADTheme.techGray)

                            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                                SpecItem(label: "CORPORATE EMAIL", value: req.email, isEmail: true)
                                SpecItem(label: "ROLE ASSIGNMENT", value: req.role ?? "Supplier Operations")
                                SpecItem(label: "INQUIRY TYPE", value: req.inquiry_type.uppercased())
                                SpecItem(label: "NETWORK FLEET / SITES", value: req.fleet_size ?? "Not specified")
                                SpecItem(label: "SUBMISSION TIMESTAMP", value: req.displayDate)
                                SpecItem(label: "SOURCE CHANNEL", value: req.source_page ?? "Website")
                            }
                        }
                        .padding(14)
                        .cadCard()

                        // 3. Operational Message from Client
                        if let msg = req.message, !msg.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Text("CLIENT INQUIRY / OPERATIONAL REQUIREMENTS")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.techGray)

                                Text(msg)
                                    .font(.system(size: 13))
                                    .lineSpacing(4)
                                    .foregroundColor(Color.white.opacity(0.9))
                                    .padding(14)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .background(SwissCADTheme.panelDark)
                                    .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))
                            }
                        }

                        // 4. ANSWER CLIENT (Dedicated Response Studio)
                        VStack(alignment: .leading, spacing: 12) {
                            HStack {
                                Text("ANSWER CLIENT // OUTBOUND RESPONSE")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.voltLime)

                                Spacer()

                                Text("DIRECT RESEND / SMTP")
                                    .font(.system(size: 9, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.techGray)
                            }

                            // Quick Templates
                            HStack(spacing: 6) {
                                QuickTemplateButton(title: "DEMO INVITE") {
                                    replySubject = "Pegasus Platform Walkthrough // \(req.name)"
                                    replyMessage = "Hi \(req.name.components(separatedBy: " ").first ?? req.name),\n\nThank you for reaching out to Pegasus regarding your \(req.fleet_size ?? "network") operations.\n\nWe'd like to schedule a 30-minute interactive walkthrough to demonstrate cross-docking, real-time driver dispatch, and our multi-role architecture. Are you available for a brief session later this week?\n\nBest regards,\nPegasus Executive Operations"
                                }

                                QuickTemplateButton(title: "PILOT ONBOARDING") {
                                    replySubject = "Pegasus Sandbox Pilot Program // \(req.company ?? req.name)"
                                    replyMessage = "Hi \(req.name.components(separatedBy: " ").first ?? req.name),\n\nWe've reviewed your operational requirements for \(req.company ?? "your organization"). We can configure an isolated sandbox cell for your sites to test live order lifecycle dispatch and warehouse replenishment.\n\nCould you confirm your target rollout timeline?\n\nBest regards,\nPegasus Architecture Team"
                                }

                                QuickTemplateButton(title: "SPEC INQUIRY") {
                                    replySubject = "Technical Specifications Clarification // Pegasus"
                                    replyMessage = "Hi \(req.name),\n\nTo tailor your platform configuration, could you share a bit more detail about your daily manifest volume and existing ERP/WMS systems?\n\nLooking forward to speaking soon,\nPegasus Solutions Lead"
                                }
                            }

                            // Subject Input
                            VStack(alignment: .leading, spacing: 4) {
                                Text("REPLY SUBJECT")
                                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.techGray)
                                TextField("Email Subject...", text: $replySubject)
                                    .textFieldStyle(.plain)
                                    .font(.system(size: 12))
                                    .padding(8)
                                    .background(SwissCADTheme.panelDark)
                                    .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))
                            }

                            // Message Editor
                            VStack(alignment: .leading, spacing: 4) {
                                Text("REPLY MESSAGE")
                                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.techGray)
                                TextEditor(text: $replyMessage)
                                    .font(.system(size: 12))
                                    .frame(minHeight: 120)
                                    .scrollContentBackground(.hidden)
                                    .padding(8)
                                    .background(SwissCADTheme.panelDark)
                                    .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))
                            }

                            if let notice = replySuccessNotice {
                                Text(notice)
                                    .font(.system(size: 11, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.emeraldGreen)
                            }
                            if let error = replyErrorNotice {
                                Text(error)
                                    .font(.system(size: 11, design: .monospaced))
                                    .foregroundColor(SwissCADTheme.laserOrange)
                            }

                            // Send Answer Buttons
                            HStack(spacing: 10) {
                                Button(action: sendClientAnswer) {
                                    HStack {
                                        if isSendingReply {
                                            ProgressView().controlSize(.small)
                                        } else {
                                            Image(systemName: "paperplane.fill")
                                        }
                                        Text(isSendingReply ? "TRANSMITTING..." : "SEND ANSWER TO CLIENT →")
                                    }
                                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                                    .foregroundColor(.black)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 10)
                                    .background(SwissCADTheme.electricCyan)
                                }
                                .buttonStyle(.plain)
                                .disabled(isSendingReply || replySubject.isEmpty || replyMessage.isEmpty)

                                if let encodedSub = replySubject.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed),
                                   let encodedBody = replyMessage.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed),
                                   let mailURL = URL(string: "mailto:\(req.email)?subject=\(encodedSub)&body=\(encodedBody)") {
                                    Link(destination: mailURL) {
                                        HStack {
                                            Image(systemName: "envelope.badge")
                                            Text("SYSTEM MAIL")
                                        }
                                        .font(.system(size: 11, weight: .bold, design: .monospaced))
                                        .foregroundColor(SwissCADTheme.techGray)
                                        .padding(.horizontal, 14)
                                        .padding(.vertical, 10)
                                        .background(SwissCADTheme.panelDark)
                                        .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))
                                    }
                                }
                            }
                        }
                        .padding(14)
                        .cadCard()

                        // 5. Workflow State Transition
                        VStack(alignment: .leading, spacing: 10) {
                            Text("WORKFLOW STATUS ADVANCEMENT")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundColor(SwissCADTheme.techGray)

                            HStack(spacing: 8) {
                                ForEach(["new", "in_review", "contacted", "qualified", "closed"], id: \.self) { st in
                                    Button(action: {
                                        onStatusChange(req.id, st)
                                    }) {
                                        Text(st.replacingOccurrences(of: "_", with: " ").uppercased())
                                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                                            .foregroundColor(req.status == st ? .black : SwissCADTheme.techGray)
                                            .padding(.horizontal, 10)
                                            .padding(.vertical, 6)
                                            .background(req.status == st ? Color.white : SwissCADTheme.panelDark)
                                            .overlay(Rectangle().stroke(req.status == st ? Color.white : SwissCADTheme.borderStroke, lineWidth: 1))
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }

                        // 6. Delete Record
                        Button(action: {
                            onDelete(req.id)
                        }) {
                            HStack {
                                Image(systemName: "trash")
                                Text("DELETE REQUEST FROM DATABASE")
                            }
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundColor(SwissCADTheme.laserOrange)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(SwissCADTheme.panelDark)
                            .overlay(Rectangle().stroke(SwissCADTheme.laserOrange.opacity(0.3), lineWidth: 1))
                        }
                        .buttonStyle(.plain)
                    }
                    .padding(20)
                }
                .background(SwissCADTheme.pitchBlack)
                .onAppear {
                    loadDefaults(for: req)
                }
                .onChange(of: req.id) {
                    loadDefaults(for: req)
                }
            } else {
                VStack(spacing: 16) {
                    Spacer()
                    SwissCADTheme.logoImage
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .frame(width: 72, height: 72)
                        .opacity(0.35)
                    Text("PEGASUS // REVIEW & ANSWER CONSOLE")
                        .font(.system(size: 11, weight: .bold, design: .monospaced))
                        .foregroundColor(SwissCADTheme.techGray)
                    Text("Select a client inquiry from the queue to inspect operational specs and draft a response.")
                        .font(.system(size: 12))
                        .foregroundColor(SwissCADTheme.mutedGray)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(SwissCADTheme.pitchBlack)
            }
        }
    }

    private func loadDefaults(for req: EnterpriseRequest) {
        replySubject = "Re: Pegasus \(req.inquiry_type.capitalized) Inquiry // \(req.name)"
        replyMessage = "Hi \(req.name.components(separatedBy: " ").first ?? req.name),\n\nThank you for submitting your request on the Pegasus website. We've reviewed your operational profile and would be glad to coordinate the next steps with your team.\n\nBest regards,\nPegasus Operations Team"
        replySuccessNotice = nil
        replyErrorNotice = nil
    }

    private func sendClientAnswer() {
        guard let req = request else { return }
        isSendingReply = true
        replySuccessNotice = nil
        replyErrorNotice = nil

        Task {
            do {
                let ok = try await EnterpriseAPIClient.shared.sendClientReply(
                    requestId: req.id,
                    recipient: req.email,
                    subject: replySubject,
                    message: replyMessage
                )
                if ok {
                    replySuccessNotice = "✓ Answer transmitted to client (\(req.email)). Status updated to Contacted."
                    onAnswerSent()
                } else {
                    replyErrorNotice = "✗ Could not deliver via API. Please use the System Mail fallback."
                }
            } catch {
                replyErrorNotice = "✗ \(error.localizedDescription)"
            }
            isSendingReply = false
        }
    }
}

private struct QuickTemplateButton: View {
    let title: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundColor(SwissCADTheme.techGray)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(SwissCADTheme.panelDark)
                .overlay(Rectangle().stroke(SwissCADTheme.borderStroke, lineWidth: 1))
        }
        .buttonStyle(.plain)
    }
}

private struct SpecItem: View {
    let label: String
    let value: String
    var isEmail: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(label)
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundColor(SwissCADTheme.techGray)

            if isEmail, let url = URL(string: "mailto:\(value)") {
                Link(value, destination: url)
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundColor(SwissCADTheme.electricCyan)
            } else {
                Text(value)
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundColor(.white)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}
