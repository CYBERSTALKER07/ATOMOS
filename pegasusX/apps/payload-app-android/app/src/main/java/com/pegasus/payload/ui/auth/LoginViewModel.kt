package com.pegasus.payload.ui.auth

import android.app.Activity
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.payload.data.auth.FirebaseAuthHelper
import com.pegasus.payload.data.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

enum class LoginMode {
    Otp,
    PinDev,
}

data class LoginUiState(
    val mode: LoginMode = LoginMode.Otp,
    val phone: String = "",
    val otpCode: String = "",
    val pin: String = "",
    val otpSent: Boolean = false,
    val loading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepository: AuthRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = _state.asStateFlow()

    fun setMode(mode: LoginMode) {
        _state.value = _state.value.copy(mode = mode, error = null, otpSent = false, otpCode = "", pin = "")
    }

    fun onPhoneChange(value: String) {
        _state.value = _state.value.copy(phone = value, error = null)
    }

    fun onOtpChange(value: String) {
        if (value.length <= 6 && value.all { it.isDigit() }) {
            _state.value = _state.value.copy(otpCode = value, error = null)
        }
    }

    fun onPinChange(value: String) {
        if (value.length <= 8 && value.all { it.isDigit() }) {
            _state.value = _state.value.copy(pin = value, error = null)
        }
    }

    fun sendOtp(activity: Activity) {
        val s = _state.value
        if (s.phone.isBlank()) {
            _state.value = s.copy(error = "Phone number required")
            return
        }
        _state.value = s.copy(loading = true, error = null)
        viewModelScope.launch {
            runCatching { FirebaseAuthHelper.sendPhoneVerification(activity, s.phone.trim()) }
                .onSuccess {
                    _state.value = _state.value.copy(loading = false, otpSent = true)
                    if (FirebaseAuthHelper.hasAutoCredential()) {
                        verifyOtp()
                    }
                }
                .onFailure { e ->
                    _state.value = _state.value.copy(
                        loading = false,
                        error = e.message ?: "Failed to send verification code",
                    )
                }
        }
    }

    fun verifyOtp() {
        val s = _state.value
        if (!FirebaseAuthHelper.hasAutoCredential() && s.otpCode.length < 6) {
            _state.value = s.copy(error = "Enter the 6-digit verification code")
            return
        }
        _state.value = s.copy(loading = true, error = null)
        viewModelScope.launch {
            runCatching {
                val idToken = FirebaseAuthHelper.verifySmsCode(s.otpCode)
                authRepository.loginWithIdToken(idToken).getOrThrow()
            }
                .onFailure { e ->
                    _state.value = _state.value.copy(
                        loading = false,
                        error = e.message ?: "Verification failed",
                    )
                }
                .onSuccess {
                    _state.value = _state.value.copy(loading = false)
                }
        }
    }

    fun submitPin() {
        val s = _state.value
        if (s.phone.isBlank() || s.pin.length < 6) {
            _state.value = s.copy(error = "Phone and PIN required")
            return
        }
        _state.value = s.copy(loading = true, error = null)
        viewModelScope.launch {
            authRepository.login(s.phone.trim(), s.pin)
                .onFailure { e ->
                    _state.value = _state.value.copy(
                        loading = false,
                        error = e.message ?: "Login failed",
                    )
                }
                .onSuccess {
                    _state.value = _state.value.copy(loading = false)
                }
        }
    }
}
