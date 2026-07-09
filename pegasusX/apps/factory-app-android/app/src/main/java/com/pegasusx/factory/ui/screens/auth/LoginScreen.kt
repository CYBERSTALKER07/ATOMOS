package com.pegasusx.factory.ui.screens.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.data.model.LoginRequest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FirebaseAuthHelper
import com.pegasusx.factory.data.remote.TokenHolder
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private enum class LoginMode {
    Otp,
    PasswordDev,
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoginScreen(
    api: FactoryApi,
    onLoginSuccess: () -> Unit,
) {
    var mode by remember { mutableStateOf(LoginMode.Otp) }
    var phone by remember { mutableStateOf("+998") }
    var otpCode by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }
    var otpSent by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val focusManager = LocalFocusManager.current
    val context = LocalContext.current
    val activity = context as? android.app.Activity

    LaunchedEffect(Unit) {
        FirebaseAuthHelper.init(context)
    }

    fun applyAuth(token: String, refreshToken: String, factoryId: String) {
        TokenHolder.token = token
        TokenHolder.refreshToken = refreshToken
        TokenHolder.factoryId = factoryId
        onLoginSuccess()
    }

    fun handleLoginFailure(code: Int) {
        error = when (code) {
            401 -> "Invalid credentials"
            403 -> "Account deactivated"
            404 -> "User not found"
            else -> "Login failed ($code)"
        }
    }

    fun verifyOtp() {
        if (!FirebaseAuthHelper.hasAutoCredential() && otpCode.length < 6) {
            error = "Enter the 6-digit verification code"
            return
        }
        loading = true
        error = null
        scope.launch {
            try {
                val idToken = FirebaseAuthHelper.verifySmsCode(otpCode)
                val resp = api.login(LoginRequest(idToken = idToken))
                if (resp.isSuccessful && resp.body() != null) {
                    val auth = resp.body()!!
                    applyAuth(auth.token, auth.refreshToken, auth.factoryId)
                } else {
                    handleLoginFailure(resp.code())
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    fun sendOtp() {
        if (phone.trim().length < 5) {
            error = "Phone number required"
            return
        }
        val act = activity
        if (act == null) {
            error = "Unable to start phone verification"
            return
        }
        loading = true
        error = null
        scope.launch {
            try {
                FirebaseAuthHelper.sendPhoneVerification(act, phone.trim())
                otpSent = true
                if (FirebaseAuthHelper.hasAutoCredential()) {
                    verifyOtp()
                }
            } catch (e: Exception) {
                error = e.message ?: "Failed to send verification code"
            } finally {
                if (!FirebaseAuthHelper.hasAutoCredential()) {
                    loading = false
                }
            }
        }
    }

    fun doPasswordLogin() {
        if (phone.isBlank() || password.isBlank()) {
            error = "Phone and password are required"
            return
        }
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.login(LoginRequest(phone = phone.trim(), password = password))
                if (resp.isSuccessful && resp.body() != null) {
                    val auth = resp.body()!!
                    applyAuth(auth.token, auth.refreshToken, auth.factoryId)
                } else {
                    handleLoginFailure(resp.code())
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    fun switchMode(next: LoginMode) {
        mode = next
        error = null
        otpSent = false
        otpCode = ""
        password = ""
        FirebaseAuthHelper.resetFlow()
    }

    Scaffold { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .imePadding()
                .padding(horizontal = PegasusSpacing.xl),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "Pegasus Factory",
                style = MaterialTheme.typography.headlineLarge,
                color = MaterialTheme.colorScheme.onSurface,
            )

            Spacer(Modifier.height(PegasusSpacing.sm))

            Text(
                text = if (mode == LoginMode.Otp) {
                    "Sign in with your registered phone number."
                } else {
                    "Dev login with phone and password when Firebase OTP is unavailable."
                },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )

            Spacer(Modifier.height(PegasusSpacing.xxl))

            OutlinedTextField(
                value = phone,
                onValueChange = { phone = it },
                label = { Text("Phone") },
                leadingIcon = { Icon(Icons.Default.Phone, contentDescription = null) },
                singleLine = true,
                enabled = !loading && (mode == LoginMode.PasswordDev || !otpSent),
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Phone,
                    imeAction = ImeAction.Next,
                ),
                keyboardActions = KeyboardActions(
                    onNext = { focusManager.moveFocus(FocusDirection.Down) },
                ),
                modifier = Modifier.fillMaxWidth(),
            )

            if (mode == LoginMode.Otp && otpSent) {
                Spacer(Modifier.height(PegasusSpacing.lg))
                OutlinedTextField(
                    value = otpCode,
                    onValueChange = {
                        if (it.length <= 6 && it.all { c -> c.isDigit() }) otpCode = it
                    },
                    label = { Text("Verification Code") },
                    singleLine = true,
                    enabled = !loading,
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.NumberPassword,
                        imeAction = ImeAction.Done,
                    ),
                    keyboardActions = KeyboardActions(onDone = { verifyOtp() }),
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            if (mode == LoginMode.PasswordDev) {
                Spacer(Modifier.height(PegasusSpacing.lg))
                OutlinedTextField(
                    value = password,
                    onValueChange = { password = it },
                    label = { Text("Password") },
                    singleLine = true,
                    visualTransformation = if (passwordVisible) {
                        VisualTransformation.None
                    } else {
                        PasswordVisualTransformation()
                    },
                    trailingIcon = {
                        IconButton(onClick = { passwordVisible = !passwordVisible }) {
                            Icon(
                                imageVector = if (passwordVisible) {
                                    Icons.Default.Visibility
                                } else {
                                    Icons.Default.VisibilityOff
                                },
                                contentDescription = if (passwordVisible) "Hide password" else "Show password",
                            )
                        }
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Done,
                    ),
                    keyboardActions = KeyboardActions(onDone = { doPasswordLogin() }),
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            if (error != null) {
                Spacer(Modifier.height(PegasusSpacing.sm))
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Warning,
                    message = error!!,
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            Spacer(Modifier.height(PegasusSpacing.xl))

            when (mode) {
                LoginMode.Otp -> {
                    if (!otpSent) {
                        Button(
                            onClick = { sendOtp() },
                            enabled = !loading && phone.isNotBlank(),
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(52.dp),
                        ) {
                            if (loading) {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(20.dp),
                                    strokeWidth = 2.dp,
                                    color = MaterialTheme.colorScheme.onPrimary,
                                )
                            } else {
                                Text("Send Code")
                            }
                        }
                    } else {
                        Button(
                            onClick = { verifyOtp() },
                            enabled = !loading && (
                                FirebaseAuthHelper.hasAutoCredential() || otpCode.length >= 6
                                ),
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(52.dp),
                        ) {
                            if (loading) {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(20.dp),
                                    strokeWidth = 2.dp,
                                    color = MaterialTheme.colorScheme.onPrimary,
                                )
                            } else {
                                Text("Sign In")
                            }
                        }
                        if (otpSent) {
                            TextButton(
                                onClick = {
                                    otpSent = false
                                    otpCode = ""
                                    FirebaseAuthHelper.resetFlow()
                                },
                                enabled = !loading,
                            ) {
                                Text("Resend code")
                            }
                        }
                    }
                }
                LoginMode.PasswordDev -> {
                    Button(
                        onClick = { doPasswordLogin() },
                        enabled = !loading && phone.isNotBlank() && password.isNotBlank(),
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(52.dp),
                    ) {
                        if (loading) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(20.dp),
                                strokeWidth = 2.dp,
                                color = MaterialTheme.colorScheme.onPrimary,
                            )
                        } else {
                            Text("Sign In")
                        }
                    }
                }
            }

            Spacer(Modifier.height(PegasusSpacing.sm))

            OutlinedButton(
                onClick = {
                    switchMode(if (mode == LoginMode.Otp) LoginMode.PasswordDev else LoginMode.Otp)
                },
                enabled = !loading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    if (mode == LoginMode.Otp) {
                        "Use password (dev)"
                    } else {
                        "Use phone OTP"
                    },
                )
            }
        }
    }
}
