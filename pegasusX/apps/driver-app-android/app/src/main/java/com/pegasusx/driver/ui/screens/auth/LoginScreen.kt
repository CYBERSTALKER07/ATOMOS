package com.pegasusx.driver.ui.screens.auth

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Phone
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.google.firebase.messaging.FirebaseMessaging
import com.pegasusx.driver.data.model.AuthResponse
import com.pegasusx.driver.data.model.LoginRequest
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.FirebaseAuthHelper
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.ui.components.DriverStateKind
import com.pegasusx.driver.ui.components.DriverStatePane
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

private enum class LoginMode {
    Otp,
    PinDev,
}

@Composable
fun LoginScreen(
    api: DriverApi,
    onLoginSuccess: () -> Unit
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

    LaunchedEffect(Unit) {
        FirebaseAuthHelper.init(context)
    }

    fun applyLoginResponse(response: AuthResponse) {
        TokenHolder.token = response.token
        TokenHolder.userId = response.userId
        TokenHolder.driverName = response.name
        TokenHolder.vehicleType = response.vehicleType
        TokenHolder.licensePlate = response.licensePlate
        TokenHolder.vehicleId = response.vehicleId
        TokenHolder.vehicleClass = response.vehicleClass
        TokenHolder.maxVolumeVU = response.maxVolumeVU
        TokenHolder.warehouseId = response.warehouseId
        TokenHolder.warehouseName = response.warehouseName
        TokenHolder.warehouseLat = response.warehouseLat
        TokenHolder.warehouseLng = response.warehouseLng
        TokenHolder.homeNodeType = response.homeNodeType
        TokenHolder.homeNodeId = response.homeNodeId
        TokenHolder.driverMode = response.driverMode
        TokenHolder.factoryId = response.factoryId
        TokenHolder.factoryName = response.factoryName
        TokenHolder.factoryLat = response.factoryLat
        TokenHolder.factoryLng = response.factoryLng
    }

    fun registerPushBestEffort() {
        scope.launch {
            try {
                val fcmToken = FirebaseMessaging.getInstance().token.await()
                api.registerDeviceToken(mapOf("token" to fcmToken, "platform" to "android"))
            } catch (_: Exception) {
            }
        }
    }

    fun finishLogin(response: AuthResponse) {
        applyLoginResponse(response)
        scope.launch {
            if (response.firebaseToken.isNotBlank()) {
                val fbIdToken = FirebaseAuthHelper.exchangeCustomToken(response.firebaseToken)
                if (fbIdToken != null) {
                    TokenHolder.firebaseIdToken = fbIdToken
                }
            }
            registerPushBestEffort()
            onLoginSuccess()
        }
    }

    fun handleLoginError(e: Exception) {
        error = when (e) {
            is retrofit2.HttpException -> when (e.code()) {
                401 -> "Invalid phone or PIN"
                403 -> "Account deactivated"
                else -> "Login failed (${e.code()})"
            }
            else -> e.message ?: "Network error. Check connection."
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
                val response = api.login(LoginRequest(idToken = idToken))
                finishLogin(response)
            } catch (e: Exception) {
                handleLoginError(e)
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

    fun doPinLogin() {
        if (phone.length < 5 || pin.isEmpty()) {
            error = "Phone and PIN are required"
            return
        }
        loading = true
        error = null
        scope.launch {
            try {
                val response = api.login(LoginRequest(phone = phone.trim(), pin = pin.trim()))
                finishLogin(response)
            } catch (e: Exception) {
                handleLoginError(e)
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
        pin = ""
    }

    Scaffold { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .imePadding(),
            contentAlignment = Alignment.Center
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.LocalShipping,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(72.dp)
                )

                Text(
                    text = stringResource(R.string.auth_login_title),
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface
                )

                Text(
                    text = stringResource(R.string.auth_login_driver_terminal),
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                Text(
                    text = if (mode == LoginMode.Otp) {
                        "Sign in with fleet phone OTP."
                    } else {
                        "Dev login with phone and PIN."
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center
                )

                Spacer(modifier = Modifier.height(8.dp))

                OutlinedTextField(
                    value = phone,
                    onValueChange = { phone = it },
                    label = { Text("Phone Number") },
                    leadingIcon = { Icon(Icons.Default.Phone, contentDescription = null) },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Phone,
                        imeAction = ImeAction.Next
                    ),
                    keyboardActions = KeyboardActions(
                        onNext = { focusManager.moveFocus(FocusDirection.Down) }
                    ),
                    singleLine = true,
                    enabled = !loading && (mode == LoginMode.PinDev || !otpSent),
                    modifier = Modifier.fillMaxWidth()
                )

                if (mode == LoginMode.Otp && otpSent) {
                    OutlinedTextField(
                        value = otpCode,
                        onValueChange = { if (it.length <= 6 && it.all { c -> c.isDigit() }) otpCode = it },
                        label = { Text("Verification Code") },
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.NumberPassword,
                            imeAction = ImeAction.Done
                        ),
                        keyboardActions = KeyboardActions(onDone = { verifyOtp() }),
                        singleLine = true,
                        enabled = !loading,
                        modifier = Modifier.fillMaxWidth()
                    )
                }

                if (mode == LoginMode.PinDev) {
                    OutlinedTextField(
                        value = pin,
                        onValueChange = { if (it.length <= 6) pin = it },
                        label = { Text("PIN") },
                        leadingIcon = { Icon(Icons.Default.Lock, contentDescription = null) },
                        trailingIcon = {
                            IconButton(onClick = { pinVisible = !pinVisible }) {
                                Icon(
                                    imageVector = if (pinVisible) Icons.Default.Visibility else Icons.Default.VisibilityOff,
                                    contentDescription = if (pinVisible) "Hide PIN" else "Show PIN"
                                )
                            }
                        },
                        visualTransformation = if (pinVisible) VisualTransformation.None else PasswordVisualTransformation(),
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.NumberPassword,
                            imeAction = ImeAction.Done
                        ),
                        keyboardActions = KeyboardActions(onDone = { doPinLogin() }),
                        singleLine = true,
                        enabled = !loading,
                        modifier = Modifier.fillMaxWidth()
                    )
                }

                error?.let {
                    DriverStatePane(
                        kind = DriverStateKind.AuthFailure,
                        headline = "Login failed",
                        body = it,
                        compact = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                when (mode) {
                    LoginMode.Otp -> {
                        if (!otpSent) {
                            Button(
                                onClick = { sendOtp() },
                                enabled = !loading && phone.isNotBlank(),
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(52.dp)
                            ) {
                                if (loading) {
                                    CircularProgressIndicator(
                                        modifier = Modifier.size(20.dp),
                                        strokeWidth = 2.dp,
                                        color = MaterialTheme.colorScheme.onPrimary
                                    )
                                } else {
                                    Text("Send Code", style = MaterialTheme.typography.labelLarge)
                                }
                            }
                        } else {
                            Button(
                                onClick = { verifyOtp() },
                                enabled = !loading && (FirebaseAuthHelper.hasAutoCredential() || otpCode.length >= 6),
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(52.dp)
                            ) {
                                if (loading) {
                                    CircularProgressIndicator(
                                        modifier = Modifier.size(20.dp),
                                        strokeWidth = 2.dp,
                                        color = MaterialTheme.colorScheme.onPrimary
                                    )
                                } else {
                                    Text("Verify & Sign In", style = MaterialTheme.typography.labelLarge)
                                }
                            }
                            OutlinedButton(
                                onClick = { sendOtp() },
                                enabled = !loading,
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Text("Resend code")
                            }
                        }
                    }
                    LoginMode.PinDev -> {
                        Button(
                            onClick = { doPinLogin() },
                            enabled = !loading,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(52.dp)
                        ) {
                            if (loading) {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(20.dp),
                                    strokeWidth = 2.dp,
                                    color = MaterialTheme.colorScheme.onPrimary
                                )
                            } else {
                                Text("Sign In", style = MaterialTheme.typography.labelLarge)
                            }
                        }
                    }
                }

                TextButton(
                    onClick = {
                        switchMode(if (mode == LoginMode.Otp) LoginMode.PinDev else LoginMode.Otp)
                    },
                    enabled = !loading
                ) {
                    Text(
                        if (mode == LoginMode.Otp) "Use PIN (dev)" else "Use phone OTP"
                    )
                }

                Text(
                    text = stringResource(R.string.mobile_driver_ui_contact_your_supplier_admin_for_credentials),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center
                )
            }
        }
    }
}
