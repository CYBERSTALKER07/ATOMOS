content = File.read("WarehouseApp/Views/Dispatch/DispatchView.swift")

rescue_sheet = <<~SWIFT
            .sheet(isPresented: $showRescueDialog) {
                NavigationStack {
                    Form {
                        if earlyCompleteReq == nil {
                            Section("Driver Lookup") {
                                TextField("Driver ID", text: $lookupDriverId)
                                Button("Find Driver") {
                                    lookupLoading = true
                                    lookupError = nil
                                    Task {
                                        defer { lookupLoading = false }
                                        do {
                                            let res = try await WarehouseOperationsService.getEarlyCompleteRequest(driverId: lookupDriverId)
                                            earlyCompleteReq = res
                                        } catch {
                                            lookupError = error.localizedDescription
                                        }
                                    }
                                }
                                .disabled(lookupDriverId.isEmpty || lookupLoading)
                                
                                if lookupLoading { ProgressView() }
                                if let err = lookupError { Text(err).foregroundColor(.red) }
                            }
                        } else if let req = earlyCompleteReq {
                            Section("Rescue Request") {
                                LabeledContent("Status", value: "\\(req["status"] ?? "")")
                                LabeledContent("Reason", value: "\\(req["reason"] ?? "")")
                                LabeledContent("Driver", value: "\\(lookupDriverId)")
                            }
                            
                            if !rescueMutating && rescueAction == nil {
                                Section("Action") {
                                    Button("Cancel Route", role: .destructive) { rescueAction = "CANCEL" }
                                    Button("Reschedule Orders") { rescueAction = "RESCHEDULE" }
                                }
                            }
                            
                            if rescueAction == "RESCHEDULE" {
                                Section("Reschedule Details") {
                                    TextField("New Start (e.g. 2026-08-01T09:00:00Z)", text: $rescueWindowStart)
                                    TextField("New End (e.g. 2026-08-01T17:00:00Z)", text: $rescueWindowEnd)
                                }
                            }
                            
                            if let action = rescueAction {
                                Section {
                                    Button("Confirm \\(action)") {
                                        rescueMutating = true
                                        Task {
                                            defer { rescueMutating = false }
                                            do {
                                                let s = rescueWindowStart.isEmpty ? nil : rescueWindowStart
                                                let e = rescueWindowEnd.isEmpty ? nil : rescueWindowEnd
                                                _ = try await WarehouseOperationsService.approveEarlyComplete(driverId: lookupDriverId, action: action, newWindowStart: s, newWindowEnd: e)
                                                showRescueDialog = false
                                                earlyCompleteReq = nil
                                                rescueAction = nil
                                                load()
                                            } catch {
                                                lookupError = error.localizedDescription
                                            }
                                        }
                                    }
                                    .disabled(rescueMutating)
                                    
                                    Button("Cancel") { rescueAction = nil }
                                        .disabled(rescueMutating)
                                }
                            }
                        }
                    }
                    .navigationTitle("Driver Rescue")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Close") { showRescueDialog = false }
                        }
                    }
                }
                .presentationDetents([.medium, .large])
            }
SWIFT

parts = content.split("                .presentationDetents([.medium])\n            }\n    }")
if parts.length == 2
  new_content = parts[0] + "                .presentationDetents([.medium])\n            }\n" + rescue_sheet + "    }" + parts[1]
  File.write("WarehouseApp/Views/Dispatch/DispatchView.swift", new_content)
else
  puts "Failed to split"
end
