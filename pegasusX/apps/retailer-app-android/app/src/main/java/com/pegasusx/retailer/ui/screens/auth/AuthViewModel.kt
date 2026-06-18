package com.pegasusx.retailer.ui.screens.auth

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.LoginRequest
import com.pegasusx.retailer.data.model.RegisterRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class AuthLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

@HiltViewModel
class AuthViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    @ApplicationContext private val context: Context,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AuthUiState())
    val uiState = _uiState.asStateFlow()

    val isAuthenticated: Boolean get() = tokenManager.getToken() != null

    // ── Login ──

    fun login(phone: String, password: String) {
        val formatted = formatUzPhone(phone)
        if (formatted == null) {
            _uiState.value = _uiState.value.copy(error = "Invalid number. Use +998 XX XXX XX XX.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        if (password.length < 4) {
            _uiState.value = _uiState.value.copy(error = "Password too short.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.login(LoginRequest(phoneNumber = formatted, password = password))
                completeAuth(response)
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "Login failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    fun loginWithOtp(idToken: String) {
        if (idToken.isBlank()) {
            _uiState.value = _uiState.value.copy(error = "Verification required.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.login(LoginRequest(idToken = idToken))
                completeAuth(response)
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "OTP login failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    private suspend fun completeAuth(response: com.pegasusx.retailer.data.model.AuthResponse) {
        tokenManager.saveToken(response.token)
        tokenManager.saveUserId(response.user.id)
        tokenManager.saveUserName(response.user.name)
        if (response.firebaseToken.isNotBlank()) {
            val fbIdToken = com.pegasusx.retailer.data.auth.FirebaseAuthHelper.exchangeCustomToken(response.firebaseToken)
            if (fbIdToken != null) tokenManager.saveFirebaseIdToken(fbIdToken)
        }
        _uiState.value = _uiState.value.copy(isLoading = false, isAuthenticated = true, error = null, loadIssue = null)
    }

    suspend fun resolveAddress(lat: Double, lng: Double): String? =
        runCatching { api.reverseGeocode(lat, lng).address.takeIf { it.isNotBlank() } }.getOrNull()

    // ── Register ──

    fun register(
        phone: String,
        password: String,
        storeName: String,
        ownerName: String,
        addressText: String,
        taxId: String?,
        latitude: Double = 0.0,
        longitude: Double = 0.0,
        receivingWindowOpen: String? = null,
        receivingWindowClose: String? = null,
        accessType: String? = null,
        storageCeilingHeightCM: Double? = null,
    ) {
        val formatted = formatUzPhone(phone)
        if (formatted == null) {
            _uiState.value = _uiState.value.copy(error = "Invalid number. Use +998 XX XXX XX XX.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        if (password.length < 4) {
            _uiState.value = _uiState.value.copy(error = "Password too short.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.register(RegisterRequest(
                    phoneNumber = formatted, password = password,
                    storeName = storeName, ownerName = ownerName,
                    addressText = addressText,
                    deliveryAddress = addressText.takeIf { it.isNotBlank() },
                    latitude = latitude, longitude = longitude,
                    taxId = taxId?.takeIf { it.isNotBlank() },
                    receivingWindowOpen = receivingWindowOpen?.takeIf { it.isNotBlank() },
                    receivingWindowClose = receivingWindowClose?.takeIf { it.isNotBlank() },
                    accessType = accessType?.takeIf { it.isNotBlank() },
                    storageCeilingHeightCM = storageCeilingHeightCM,
                ))
                tokenManager.saveToken(response.token)
                tokenManager.saveUserId(response.user.id)
                tokenManager.saveUserName(response.user.name)
                // Exchange Firebase custom token (graceful degradation)
                if (response.firebaseToken.isNotBlank()) {
                    // Firebase OTP scaffold: exchange custom token when configured.
                    // Full Firebase phone OTP remains behind env/feature gate — not required for retailer login.
                    val fbIdToken = com.pegasusx.retailer.data.auth.FirebaseAuthHelper.exchangeCustomToken(response.firebaseToken)
                    if (fbIdToken != null) tokenManager.saveFirebaseIdToken(fbIdToken)
                }
                try {
                    api.setupRetailer(
                        body = mapOf(
                            "store_name" to storeName,
                            "owner_name" to ownerName,
                            "address_text" to addressText,
                            "latitude" to latitude,
                            "longitude" to longitude,
                        ),
                        idempotencyKey = "retailer-setup:${response.user.id}",
                    )
                } catch (_: Exception) {
                    // Registration already captured core profile; setup is additive best-effort.
                }
                _uiState.value = _uiState.value.copy(isLoading = false, isAuthenticated = true, error = null, loadIssue = null)
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "Registration failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    fun logout() {
        tokenManager.clearToken()
        com.pegasusx.retailer.data.auth.FirebaseAuthHelper.signOut()
        _uiState.value = AuthUiState()
    }

    fun clearError() {
        _uiState.value = _uiState.value.copy(error = null, loadIssue = null)
    }

    private fun resolveLoadIssue(error: Exception): AuthLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> AuthLoadIssue.RESTRICTED
            error is IOException -> AuthLoadIssue.OFFLINE
            else -> AuthLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: AuthLoadIssue, fallback: String): String {
        return when (issue) {
            AuthLoadIssue.RESTRICTED -> "Access is restricted for this account"
            AuthLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            AuthLoadIssue.ERROR -> error.message ?: fallback
        }
    }

    private fun formatUzPhone(raw: String): String? {
        val digits = raw.replace("[^0-9]".toRegex(), "")
        val phone = when {
            digits.startsWith("998") && digits.length == 12 -> "+$digits"
            digits.length == 9 -> "+998$digits"
            raw.startsWith("+998") && raw.length == 13 -> raw
            else -> null
        }
        return phone?.takeIf { it.matches(Regex("^\\+998\\d{9}$")) }
    }
}

data class AuthUiState(
    val isLoading: Boolean = false,
    val isAuthenticated: Boolean = false,
    val error: String? = null,
    val loadIssue: AuthLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            AuthLoadIssue.RESTRICTED -> "Access is restricted for this account"
            AuthLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            AuthLoadIssue.ERROR -> "Authentication request failed"
            null -> null
        }
}
