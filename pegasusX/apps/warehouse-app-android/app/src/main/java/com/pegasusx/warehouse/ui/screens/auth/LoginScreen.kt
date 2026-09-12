package com.pegasusx.warehouse.ui.screens.auth

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Phone
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.LoginRequest
import com.pegasusx.warehouse.data.push.DeviceTokenRegistrar
import com.pegasusx.warehouse.data.remote.FirebaseAuthHelper
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.portal.WarehousePortalFeature
import com.pegasusx.warehouse.ui.portal.WarehousePortalLinks
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private enum class LoginMode { Otp, PinDev }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoginScreen(
    api: WarehouseApi,
    onLoginSuccess: () -> Unit,
) {
    var mode by remember { mutableStateOf(LoginMode.Otp) }
    var phone by remember { mutableStateOf("+998") }
    var otpCode by remember { mutableStateOf("") }
    var pin by remember { mutableStateOf("") }
    var pinVisible by remember { mutableStateOf(false) }
    var otpSent by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val focusManager = LocalFocusManager.current
    val context = LocalContext.current
    val activity = context as? android.app.Activity

    LaunchedEffect(Unit) { FirebaseAuthHelper.init(context) }

    fun applyAuth(token: String, refreshToken: String, warehouseId: String) {
        TokenHolder.token = token
        TokenHolder.refreshToken = refreshToken
        TokenHolder.warehouseId = warehouseId
        DeviceTokenRegistrar.uploadBestEffort(api)
        onLoginSuccess()
    }

    fun openPortalRegister() {
        val intent = Intent(Intent.ACTION_VIEW, WarehousePortalLinks.openUri(WarehousePortalFeature.REGISTER))
        context.startActivity(intent)
    }

    Scaffold { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = PegasusSpacing.xl),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text("Pegasus Warehouse", style = MaterialTheme.typography.headlineLarge)
            Spacer(Modifier.height(PegasusSpacing.sm))
            Text(
                text = if (mode == LoginMode.Otp) "Sign in with phone OTP" else "Dev fallback: phone + PIN",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(PegasusSpacing.xxl))

            OutlinedTextField(
                value = phone,
                onValueChange = { phone = it },
                label = { Text("Phone") },
                leadingIcon = { Icon(Icons.Default.Phone, contentDescription = null) },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone, imeAction = ImeAction.Next),
                keyboardActions = KeyboardActions(onNext = { focusManager.moveFocus(FocusDirection.Down) }),
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(Modifier.height(PegasusSpacing.lg))

            if (mode == LoginMode.Otp) {
                if (otpSent) {
                    OutlinedTextField(
                        value = otpCode,
                        onValueChange = { if (it.length <= 6) otpCode = it },
                        label = { Text("Verification code") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword, imeAction = ImeAction.Done),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Spacer(Modifier.height(PegasusSpacing.lg))
                }
                if (!otpSent) {
                    Button(
                        onClick = {
                            if (activity == null) {
                                error = "OTP requires an Activity context"
                                return@Button
                            }
                            loading = true
                            error = null
                            scope.launch {
                                try {
                                    FirebaseAuthHelper.sendPhoneVerification(activity, phone)
                                    otpSent = true
                                } catch (e: Exception) {
                                    error = e.message ?: "Failed to send code"
                                } finally {
                                    loading = false
                                }
                            }
                        },
                        enabled = !loading && phone.isNotBlank(),
                        modifier = Modifier.fillMaxWidth().height(52.dp),
                    ) {
                        Text(if (loading) "Sending…" else "Send code")
                    }
                } else {
                    Button(
                        onClick = {
                            if (!FirebaseAuthHelper.hasAutoCredential() && otpCode.length < 6) {
                                error = "Enter the 6-digit verification code"
                                return@Button
                            }
                            loading = true
                            error = null
                            scope.launch {
                                try {
                                    val idToken = FirebaseAuthHelper.verifySmsCode(otpCode)
                                    val resp = api.login(LoginRequest(idToken = idToken))
                                    if (resp.isSuccessful && resp.body() != null) {
                                        val auth = resp.body()!!
                                        applyAuth(auth.token, auth.refreshToken, auth.warehouseId)
                                    } else {
                                        error = when (resp.code()) {
                                            404 -> "No account found. Register your warehouse on the web portal."
                                            else -> "Login failed (${resp.code()})"
                                        }
                                    }
                                } catch (e: Exception) {
                                    error = e.message ?: "Network error"
                                } finally {
                                    loading = false
                                }
                            }
                        },
                        enabled = !loading,
                        modifier = Modifier.fillMaxWidth().height(52.dp),
                    ) {
                        if (loading) {
                            CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
                        } else {
                            Text("Verify & Sign In")
                        }
                    }
                }
            } else {
                OutlinedTextField(
                    value = pin,
                    onValueChange = { if (it.length <= 6) pin = it },
                    label = { Text("PIN") },
                    singleLine = true,
                    visualTransformation = if (pinVisible) VisualTransformation.None else PasswordVisualTransformation(),
                    trailingIcon = {
                        IconButton(onClick = { pinVisible = !pinVisible }) {
                            Icon(
                                if (pinVisible) Icons.Default.Visibility else Icons.Default.VisibilityOff,
                                contentDescription = null,
                            )
                        }
                    },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword, imeAction = ImeAction.Done),
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(PegasusSpacing.lg))
                Button(
                    onClick = {
                        loading = true
                        error = null
                        scope.launch {
                            try {
                                val resp = api.login(LoginRequest(phone = phone, pin = pin))
                                if (resp.isSuccessful && resp.body() != null) {
                                    val auth = resp.body()!!
                                    applyAuth(auth.token, auth.refreshToken, auth.warehouseId)
                                } else {
                                    error = when (resp.code()) {
                                        404 -> "No account found. Register your warehouse on the web portal."
                                        else -> "Login failed (${resp.code()})"
                                    }
                                }
                            } catch (e: Exception) {
                                error = e.message ?: "Network error"
                            } finally {
                                loading = false
                            }
                        }
                    },
                    enabled = !loading && phone.isNotBlank() && pin.length >= 4,
                    modifier = Modifier.fillMaxWidth().height(52.dp),
                ) {
                    Text("Sign In")
                }
            }

            if (error != null) {
                Spacer(Modifier.height(PegasusSpacing.sm))
                Text(error!!, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
            }

            Spacer(Modifier.height(PegasusSpacing.lg))
            OutlinedButton(
                onClick = {
                    error = null
                    otpSent = false
                    otpCode = ""
                    FirebaseAuthHelper.resetFlow()
                    mode = if (mode == LoginMode.Otp) LoginMode.PinDev else LoginMode.Otp
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (mode == LoginMode.Otp) "Use PIN (dev)" else "Use phone OTP")
            }
            TextButton(onClick = ::openPortalRegister) {
                Text("New warehouse? Register on the web portal")
            }
        }
    }
}
