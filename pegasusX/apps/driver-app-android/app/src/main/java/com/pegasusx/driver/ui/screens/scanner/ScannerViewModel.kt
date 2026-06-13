package com.pegasusx.driver.ui.screens.scanner

import android.annotation.SuppressLint
import android.app.Application
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.pegasusx.driver.data.model.ValidateQRRequest
import com.pegasusx.driver.data.model.ValidateQRResponse
import com.pegasusx.driver.data.model.VerifyHandshakeRequest
import com.pegasusx.driver.data.remote.DriverApi
import dagger.hilt.android.lifecycle.HiltViewModel
import org.json.JSONObject
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import kotlinx.coroutines.withTimeout
import javax.inject.Inject

data class ScannerUiState(
    val isScanning: Boolean = true,
    val scannedToken: String? = null,
    val isSubmitting: Boolean = false,
    val validated: ValidateQRResponse? = null,
    val handshakeVerified: Boolean = false,
    val error: String? = null
)

@HiltViewModel
class ScannerViewModel @Inject constructor(
    private val app: Application,
    private val api: DriverApi
) : ViewModel() {

    private val _state = MutableStateFlow(ScannerUiState())
    val state: StateFlow<ScannerUiState> = _state.asStateFlow()

    private val fusedClient = LocationServices.getFusedLocationProviderClient(app)

    fun onQrDetected(rawValue: String) {
        if (!_state.value.isScanning) return
        _state.value = _state.value.copy(isScanning = false, scannedToken = rawValue)
        vibrateOnDetect()
        validateQR(rawValue)
    }

    private fun vibrateOnDetect() {
        val vibrator = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            val mgr = app.getSystemService(VibratorManager::class.java)
            mgr?.defaultVibrator
        } else {
            @Suppress("DEPRECATION")
            app.getSystemService(Vibrator::class.java)
        }
        vibrator?.vibrate(VibrationEffect.createOneShot(150, VibrationEffect.DEFAULT_AMPLITUDE))
    }

    @SuppressLint("MissingPermission")
    private fun validateQR(qrToken: String) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isSubmitting = true, error = null)
            try {
                val parsedJson = runCatching { JSONObject(qrToken) }.getOrNull()
                val parsedToken = parsedJson?.optString("token")?.takeIf { it.isNotBlank() }
                val parsedOrderId = parsedJson?.optString("order_id")?.takeIf { it.isNotBlank() }

                val effectiveToken = parsedToken ?: qrToken
                val parts = effectiveToken.split(":")
                val orderId = parsedOrderId ?: if (parts.size >= 2) parts[0] else effectiveToken

                val response = withTimeout(30_000L) {
                    api.validateQR(
                        ValidateQRRequest(orderId = orderId, scannedToken = effectiveToken)
                    )
                }

                val location = fusedClient.getCurrentLocation(
                    Priority.PRIORITY_HIGH_ACCURACY,
                    CancellationTokenSource().token
                ).await()
                if (location != null) {
                    val handshake = api.verifyHandshake(
                        VerifyHandshakeRequest(
                            orderId = orderId,
                            token = effectiveToken,
                            latitude = location.latitude,
                            longitude = location.longitude,
                        )
                    )
                    if (!handshake.success) {
                        _state.value = _state.value.copy(
                            isSubmitting = false,
                            error = handshake.message.ifBlank { "Handshake verification failed" }
                        )
                        return@launch
                    }
                    _state.value = _state.value.copy(
                        isSubmitting = false,
                        validated = response,
                        handshakeVerified = true,
                    )
                } else {
                    _state.value = _state.value.copy(
                        isSubmitting = false,
                        validated = response,
                        error = "GPS unavailable — handshake not verified",
                    )
                }
            } catch (e: TimeoutCancellationException) {
                _state.value = _state.value.copy(
                    isSubmitting = false,
                    error = "QR validation timed out. Please retry."
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(
                    isSubmitting = false,
                    error = e.message ?: "QR validation failed"
                )
            }
        }
    }

    fun resetScanner() {
        _state.value = ScannerUiState()
    }
}
