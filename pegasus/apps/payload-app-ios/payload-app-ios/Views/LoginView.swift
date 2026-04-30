//
//  LoginView.swift
//  payload-app-ios
//

import SwiftUI

struct LoginView: View {
    @State private var viewModel = LoginViewModel()
    @FocusState private var focus: Field?

    private enum Field { case phone, pin }

    var body: some View {
        ZStack {
            Color(.systemGroupedBackground).ignoresSafeArea()
            VStack(spacing: 16) {
                Text("Pegasus Payload Terminal")
                    .font(.largeTitle.weight(.semibold))
                Text("Sign in with your warehouse PIN")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .padding(.bottom, 8)

                TextField("Phone", text: $viewModel.phone)
                    .keyboardType(.phonePad)
                    .textContentType(.telephoneNumber)
                    .focused($focus, equals: .phone)
                    .padding(12)
                    .background(.thinMaterial, in: .rect(cornerRadius: 12))
                    .disabled(viewModel.loading)

                SecureField("6-digit PIN", text: Binding(
                    get: { viewModel.pin },
                    set: { viewModel.setPin($0) }
                ))
                .keyboardType(.numberPad)
                .textContentType(.oneTimeCode)
                .focused($focus, equals: .pin)
                .padding(12)
                .background(.thinMaterial, in: .rect(cornerRadius: 12))
                .disabled(viewModel.loading)

                if let error = viewModel.error {
                    Text(error)
                        .font(.footnote)
                        .foregroundStyle(.red)
                }

                Button {
                    Task { await viewModel.submit() }
                } label: {
                    Group {
                        if viewModel.loading {
                            ProgressView().tint(.white)
                        } else {
                            Text("Sign In")
                                .font(.headline)
                        }
                    }
                    .frame(maxWidth: .infinity, minHeight: 48)
                }
                .buttonStyle(.borderedProminent)
                .disabled(viewModel.loading)
            }
            .padding(32)
            .frame(maxWidth: 480)
        }
        .onAppear { focus = .phone }
    }
}

#Preview { LoginView() }
