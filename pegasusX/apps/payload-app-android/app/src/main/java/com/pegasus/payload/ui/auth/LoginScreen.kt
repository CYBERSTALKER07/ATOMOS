package com.pegasus.payload.ui.auth

import android.app.Activity
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.PegasusStatePane
import com.pegasus.design.PegasusStateKind

/**
 * LoginScreen — Firebase phone OTP (primary) with PIN dev fallback.
 */
@Composable
fun LoginScreen(viewModel: LoginViewModel = hiltViewModel()) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val activity = LocalContext.current as Activity

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            Surface(
                modifier = Modifier
                    .widthIn(max = 560.dp)
                    .padding(32.dp),
                shape = MaterialTheme.shapes.extraLarge,
                color = MaterialTheme.colorScheme.surfaceContainerLow,
            ) {
                Column(
                    modifier = Modifier.padding(32.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(20.dp),
                ) {
                    Text(
                        text = "Pegasus Payload Terminal",
                        style = MaterialTheme.typography.headlineMedium,
                        color = MaterialTheme.colorScheme.onBackground,
                    )
                    Text(
                        text = if (state.mode == LoginMode.Otp) {
                            "Sign in with your warehouse phone number. We will send a one-time code."
                        } else {
                            "Dev login with phone and PIN when Firebase OTP is unavailable."
                        },
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Spacer(Modifier.height(4.dp))

                    OutlinedTextField(
                        value = state.phone,
                        onValueChange = viewModel::onPhoneChange,
                        label = { Text("Phone") },
                        singleLine = true,
                        enabled = !state.loading && (state.mode == LoginMode.PinDev || !state.otpSent),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                        modifier = Modifier.fillMaxWidth(),
                    )

                    if (state.mode == LoginMode.Otp) {
                        if (state.otpSent) {
                            OutlinedTextField(
                                value = state.otpCode,
                                onValueChange = viewModel::onOtpChange,
                                label = { Text("Verification code") },
                                singleLine = true,
                                enabled = !state.loading,
                                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
                                modifier = Modifier.fillMaxWidth(),
                            )
                        }
                    } else {
                        OutlinedTextField(
                            value = state.pin,
                            onValueChange = viewModel::onPinChange,
                            label = { Text("PIN") },
                            singleLine = true,
                            enabled = !state.loading,
                            visualTransformation = PasswordVisualTransformation(),
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }

                    state.error?.let { msg ->
                        PegasusStatePane(
                            kind = PegasusStateKind.AuthFailure,
                            headline = "Authentication Failed",
                            body = msg,
                            modifier = Modifier.fillMaxWidth().heightIn(min = 160.dp),
                            actionLabel = "Dismiss",
                            onAction = viewModel::clearError,
                        )
                    }

                    when (state.mode) {
                        LoginMode.Otp -> {
                            if (!state.otpSent) {
                                Button(
                                    onClick = { viewModel.sendOtp(activity) },
                                    enabled = !state.loading && state.phone.isNotBlank(),
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .height(56.dp),
                                ) {
                                    Text("Send Code", style = MaterialTheme.typography.titleMedium)
                                }
                            } else {
                                Button(
                                    onClick = viewModel::verifyOtp,
                                    enabled = !state.loading && state.otpCode.length >= 6,
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .height(56.dp),
                                ) {
                                    Text("Verify & Sign In", style = MaterialTheme.typography.titleMedium)
                                }
                                OutlinedButton(
                                    onClick = { viewModel.sendOtp(activity) },
                                    enabled = !state.loading,
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    Text("Resend code")
                                }
                            }
                        }
                        LoginMode.PinDev -> {
                            Button(
                                onClick = viewModel::submitPin,
                                enabled = !state.loading,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(56.dp),
                            ) {
                                Text("Sign In with PIN", style = MaterialTheme.typography.titleMedium)
                            }
                        }
                    }

                    TextButton(
                        onClick = {
                            viewModel.setMode(
                                if (state.mode == LoginMode.Otp) LoginMode.PinDev else LoginMode.Otp,
                            )
                        },
                        enabled = !state.loading,
                    ) {
                        Text(
                            if (state.mode == LoginMode.Otp) "Use PIN (dev)" else "Use phone OTP",
                        )
                    }
                }
            }
            if (state.loading) {
                Surface(modifier = Modifier.fillMaxSize(), color = androidx.compose.ui.graphics.Color.Black.copy(alpha = 0.4f)) {
                    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Surface(shape = MaterialTheme.shapes.medium, color = MaterialTheme.colorScheme.surface, modifier = Modifier.padding(32.dp)) {
                            com.pegasus.design.PegasusLoadingState(title = "Authenticating", body = "Verifying credentials with dispatch.", modifier = Modifier.padding(32.dp))
                        }
                    }
                }
            }
        }
    }
}
