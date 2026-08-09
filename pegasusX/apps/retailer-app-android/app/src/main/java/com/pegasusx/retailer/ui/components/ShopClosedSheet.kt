package com.pegasusx.retailer.ui.components

import androidx.compose.ui.res.stringResource

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CameraAlt
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.data.api.MediaUploadService
import com.pegasusx.retailer.data.api.ShopClosedAlert
import com.pegasusx.retailer.ui.theme.MotionTokens
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import kotlinx.coroutines.launch
import com.pegasusx.retailer.R

private fun shopClosedOptionLabel(option: String): String = when (option) {
    "OPEN_NOW" -> "I am open now"
    "5_MIN" -> "I will be back in 5 mins"
    "CALL_ME" -> "Call me"
    "CLOSED_TODAY" -> "Closed for the day"
    "RESCHEDULE" -> "Reschedule delivery"
    "CREDIT_LEAVE" -> "Leave on credit"
    "CANCEL" -> "Cancel remaining"
    "AUTHORIZE_BYPASS" -> "Authorize bypass offload"
    else -> option
}

@Composable
fun ShopClosedSheet(
    alert: ShopClosedAlert,
    isSubmitting: Boolean,
    errorMessage: String?,
    mediaUpload: MediaUploadService,
    onRespond: (option: String, photoUrl: String?) -> Unit,
) {
    var bypassPending by remember(alert.orderId) { mutableStateOf(false) }
    var bypassPhotoUrl by remember(alert.orderId) { mutableStateOf<String?>(null) }
    var uploading by remember { mutableStateOf(false) }
    var localError by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    val photoPicker = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.GetContent(),
    ) { uri: Uri? ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            uploading = true
            localError = null
            try {
                bypassPhotoUrl = mediaUpload.uploadJpegUri(
                    uri = uri,
                    purpose = "claim_evidence",
                    orderId = alert.orderId,
                )
            } catch (e: Exception) {
                bypassPhotoUrl = null
                localError = e.message ?: "Photo upload failed"
            } finally {
                uploading = false
            }
        }
    }

    AnimatedVisibility(
        visible = true,
        enter = fadeIn(tween(MotionTokens.DurationShort4)) +
            slideInVertically(
                animationSpec = tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate),
            ) { it / 3 },
        exit = fadeOut(tween(MotionTokens.DurationShort2)) +
            slideOutVertically(
                animationSpec = tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedAccelerate),
            ) { it / 3 },
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(MaterialTheme.colorScheme.scrim.copy(alpha = 0.45f)),
            contentAlignment = Alignment.BottomCenter,
        ) {
            Surface(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 16.dp),
                shape = SoftSquircleShape,
                color = MaterialTheme.colorScheme.surface,
                tonalElevation = 6.dp,
            ) {
                Column(
                    modifier = Modifier.padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Icon(
                        Icons.Rounded.Warning,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.tertiary,
                    )
                    Text(
                        text = stringResource(R.string.mobile_retailer_ui_driver_drivername_reported_your_shop_is_closed, alert.driverName),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = stringResource(R.string.mobile_retailer_ui_confirm_your_status_so_we_can_manage_order_takelast, alert.orderId.takeLast(6)),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    val displayError = localError ?: errorMessage
                    if (!displayError.isNullOrBlank()) {
                        Text(
                            text = displayError,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.error,
                        )
                    }
                    if (bypassPending) {
                        Text(
                            text = stringResource(R.string.mobile_retailer_ui_doorway_drop_off_proof_is_required_for_authorize_bypass),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        OutlinedButton(
                            onClick = { photoPicker.launch("image/*") },
                            enabled = !isSubmitting && !uploading,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Icon(Icons.Rounded.CameraAlt, contentDescription = null)
                            Text(
                                if (uploading) "Uploading…"
                                else if (bypassPhotoUrl != null) "Replace photo"
                                else "Take or choose photo",
                                modifier = Modifier.padding(start = 8.dp),
                            )
                        }
                        if (bypassPhotoUrl != null) {
                            Button(
                                onClick = { onRespond("AUTHORIZE_BYPASS", bypassPhotoUrl) },
                                enabled = !isSubmitting && !uploading,
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Text("Confirm bypass")
                            }
                        }
                        TextButton(
                            onClick = {
                                bypassPending = false
                                bypassPhotoUrl = null
                                localError = null
                            },
                            enabled = !isSubmitting,
                        ) {
                            Text("Cancel bypass")
                        }
                    }
                    alert.options.forEach { option ->
                        Button(
                            onClick = {
                                if (option == "AUTHORIZE_BYPASS") {
                                    bypassPending = true
                                    localError = null
                                    return@Button
                                }
                                onRespond(option, null)
                            },
                            enabled = !isSubmitting && !uploading,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            if (isSubmitting) {
                                CircularProgressIndicator(
                                    modifier = Modifier.padding(end = 8.dp),
                                    strokeWidth = 2.dp,
                                )
                            }
                            Text(shopClosedOptionLabel(option))
                        }
                    }
                }
            }
        }
    }
}
