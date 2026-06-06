package com.pegasusx.supplier.ui.screens.auth

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Phone
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.LoginRequest
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.ui.components.SupplierRuntimeBanner
import com.pegasusx.supplier.ui.components.SupplierRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoginScreen(
    api: SupplierApi,
    onLoginSuccess: () -> Unit,
) {
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val focusManager = LocalFocusManager.current

    Scaffold { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = PegasusSpacing.xl),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "Pegasus Supplier",
                style = MaterialTheme.typography.headlineLarge,
            )
            Spacer(Modifier.height(PegasusSpacing.sm))
            Text(
                text = "Sign in to manage supplier operations",
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
            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text("Password") },
                singleLine = true,
                visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
                trailingIcon = {
                    IconButton(onClick = { passwordVisible = !passwordVisible }) {
                        Icon(
                            if (passwordVisible) Icons.Default.Visibility else Icons.Default.VisibilityOff,
                            contentDescription = null,
                        )
                    }
                },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { focusManager.clearFocus() }),
                modifier = Modifier.fillMaxWidth(),
            )

            if (error != null) {
                Spacer(Modifier.height(PegasusSpacing.sm))
                SupplierRuntimeBanner(
                    tone = SupplierRuntimeTone.Warning,
                    message = error!!,
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            Spacer(Modifier.height(PegasusSpacing.xl))
            Button(
                onClick = {
                    error = null
                    loading = true
                    scope.launch {
                        try {
                            val resp = api.login(LoginRequest(phone = phone.trim(), password = password))
                            val body = resp.body()
                            if (resp.isSuccessful && body != null) {
                                val jwt = body.token
                                if (jwt.isNullOrBlank()) {
                                    error = "Login succeeded but no token returned"
                                } else {
                                    TokenHolder.token = jwt
                                    TokenHolder.refreshToken = body.refreshToken
                                    TokenHolder.supplierId = body.supplierId
                                    TokenHolder.isConfigured = body.isConfigured
                                    onLoginSuccess()
                                }
                            } else {
                                error = "Login failed (${resp.code()})"
                            }
                        } catch (e: Exception) {
                            error = e.message ?: "Network error"
                        } finally {
                            loading = false
                        }
                    }
                },
                enabled = !loading && phone.isNotBlank() && password.isNotBlank(),
                modifier = Modifier.fillMaxWidth().height(52.dp),
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
}
