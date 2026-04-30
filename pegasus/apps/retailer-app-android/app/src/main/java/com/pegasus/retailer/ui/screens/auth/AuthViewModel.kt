package com.pegasus.retailer.ui.screens.auth

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.LabApi
import com.pegasus.retailer.data.local.TokenManager
import com.pegasus.retailer.data.model.LoginRequest
import com.pegasus.retailer.data.model.RegisterRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class AuthViewModel @Inject constructor(
    private val api: LabApi,
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
            _uiState.value = _uiState.value.copy(error = "Invalid number. Use +998 XX XXX XX XX.")
            return
        }
        if (password.length < 4) {
            _uiState.value = _uiState.value.copy(error = "Password too short.")
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null)
            try {
                val response = api.login(LoginRequest(phoneNumber = formatted, password = password))
                tokenManager.saveToken(response.token)
                tokenManager.saveUserId(response.user.id)
                tokenManager.saveUserName(response.user.name)
                // Exchange Firebase custom token (graceful degradation)
                if (response.firebaseToken.isNotBlank()) {
                    val fbIdToken = com.pegasus.retailer.data.auth.FirebaseAuthHelper.exchangeCustomToken(response.firebaseToken)
                    if (fbIdToken != null) tokenManager.saveFirebaseIdToken(fbIdToken)
                }
                _uiState.value = _uiState.value.copy(isLoading = false, isAuthenticated = true)
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(isLoading = false, error = e.message ?: "Login failed")
            }
        }
    }

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
            _uiState.value = _uiState.value.copy(error = "Invalid number. Use +998 XX XXX XX XX.")
            return
        }
        if (password.length < 4) {
            _uiState.value = _uiState.value.copy(error = "Password too short.")
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null)
            try {
                val response = api.register(RegisterRequest(
                    phoneNumber = formatted, password = password,
                    storeName = storeName, ownerName = ownerName,
                    addressText = addressText, latitude = latitude, longitude = longitude,
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
                    val fbIdToken = com.pegasus.retailer.data.auth.FirebaseAuthHelper.exchangeCustomToken(response.firebaseToken)
                    if (fbIdToken != null) tokenManager.saveFirebaseIdToken(fbIdToken)
                }
                _uiState.value = _uiState.value.copy(isLoading = false, isAuthenticated = true)
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(isLoading = false, error = e.message ?: "Registration failed")
            }
        }
    }

    fun logout() {
        tokenManager.clearToken()
        com.pegasus.retailer.data.auth.FirebaseAuthHelper.signOut()
        _uiState.value = AuthUiState()
    }

    fun clearError() {
        _uiState.value = _uiState.value.copy(error = null)
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
)
