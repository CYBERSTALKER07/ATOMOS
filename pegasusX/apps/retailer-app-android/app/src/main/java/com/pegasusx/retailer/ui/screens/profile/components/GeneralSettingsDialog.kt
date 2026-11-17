package com.pegasusx.retailer.ui.screens.profile.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.OpenInNew
import androidx.compose.material.icons.rounded.ContentCopy
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.local.RetailerLanguage
import com.pegasusx.retailer.data.local.RetailerWorkspaceMode
import com.pegasusx.retailer.data.model.TelegramLinkRequestResponse
import com.pegasusx.retailer.data.model.TelegramLinkStatusResponse
import com.pegasusx.retailer.ui.theme.SquircleShape

@Composable
fun GeneralSettingsDialog(
    selectedLanguage: RetailerLanguage,
    selectedWorkspaceMode: RetailerWorkspaceMode,
    onSelectLanguage: (RetailerLanguage) -> Unit,
    onSelectWorkspaceMode: (RetailerWorkspaceMode) -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.general_settings_dialog_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = stringResource(R.string.general_settings_language_label),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                )
                RetailerLanguage.entries.forEach { language ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onSelectLanguage(language) }
                            .padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        RadioButton(
                            selected = language == selectedLanguage,
                            onClick = { onSelectLanguage(language) },
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "${language.nativeLabel} (${language.code.uppercase()})",
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }

                HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                Text(
                    text = stringResource(R.string.general_settings_workspace_label),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                )
                Text(
                    text = stringResource(R.string.general_settings_workspace_hint),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                listOf(
                    RetailerWorkspaceMode.FAST to stringResource(R.string.workspace_mode_fast),
                    RetailerWorkspaceMode.FULL to stringResource(R.string.workspace_mode_full),
                ).forEach { (mode, label) ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onSelectWorkspaceMode(mode) }
                            .padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        RadioButton(
                            selected = mode == selectedWorkspaceMode,
                            onClick = { onSelectWorkspaceMode(mode) },
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(text = label, style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }
        },
        confirmButton = {
            Button(onClick = onDismiss) {
                Text(stringResource(R.string.general_settings_save))
            }
        },
    )
}

@Composable
fun TelegramQuickBindCard(
    status: TelegramLinkStatusResponse?,
    linkRequest: TelegramLinkRequestResponse?,
    busy: Boolean,
    error: String?,
    onGenerateOtp: () -> Unit,
) {
    val clipboardManager = LocalClipboardManager.current
    val uriHandler = LocalUriHandler.current

    ElevatedCard(
        shape = SquircleShape,
        colors = CardDefaults.elevatedCardColors(containerColor = MaterialTheme.colorScheme.surface),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                text = stringResource(R.string.telegram_link_title),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = stringResource(R.string.telegram_link_subtitle),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (status?.linked == true) {
                val handle = status.username?.takeIf { it.isNotBlank() }
                    ?: status.telegramChatId.orEmpty()
                Text(
                    text = stringResource(R.string.telegram_link_linked_fmt, handle),
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.primary,
                )
            }
            if (!error.isNullOrBlank()) {
                Text(
                    text = error,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                )
            }
            if (linkRequest != null && linkRequest.otpCode.isNotBlank()) {
                Text(
                    text = stringResource(R.string.telegram_link_otp_fmt, linkRequest.otpCode),
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.SemiBold,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(
                        onClick = { clipboardManager.setText(AnnotatedString(linkRequest.otpCode)) },
                    ) {
                        Icon(Icons.Rounded.ContentCopy, contentDescription = null)
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(stringResource(R.string.telegram_link_copy_otp))
                    }
                    val deepLink = linkRequest.botDeepLink ?: linkRequest.deepLink
                    if (!deepLink.isNullOrBlank()) {
                        OutlinedButton(onClick = { runCatching { uriHandler.openUri(deepLink) } }) {
                            Icon(Icons.AutoMirrored.Rounded.OpenInNew, contentDescription = null)
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(stringResource(R.string.telegram_link_open_bot))
                        }
                    }
                }
            }
            Spacer(modifier = Modifier.height(2.dp))
            Button(
                onClick = onGenerateOtp,
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.telegram_link_generate))
            }
        }
    }
}
