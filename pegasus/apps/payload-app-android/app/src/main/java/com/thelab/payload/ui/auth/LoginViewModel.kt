package com.thelab.payload.ui.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.payload.data.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

data class LoginUiState(
    val phone: String = "",
    val pin: String = "",
    val loading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepository: AuthRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = _state.asStateFlow()

    fun onPhoneChange(value: String) {
        _state.value = _state.value.copy(phone = value, error = null)
    }

    fun onPinChange(value: String) {
        if (value.length <= 6 && value.all { it.isDigit() }) {
            _state.value = _state.value.copy(pin = value, error = null)
        }
    }

    fun submit() {
        val s = _state.value
        if (s.phone.isBlank() || s.pin.length != 6) {
            _state.value = s.copy(error = "Phone and 6-digit PIN required")
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
