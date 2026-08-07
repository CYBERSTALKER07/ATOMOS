package com.pegasusx.driver.ui.screens.manifest.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.remote.ConnectionState
import com.pegasusx.driver.ui.components.WsConnectionPill
import com.pegasusx.driver.ui.theme.PegasusSpacing

@Composable
fun ManifestHeader(
    pendingCount: Int,
    loadingMode: Boolean,
    wsConnectionState: ConnectionState,
    onToggleLoadingMode: () -> Unit,
    onRefresh: () -> Unit,
) {
    val colorScheme = MaterialTheme.colorScheme
    val typography = MaterialTheme.typography
    Column(
        modifier = Modifier
            .padding(horizontal = PegasusSpacing.s20)
            .padding(top = 60.dp, bottom = PegasusSpacing.s20)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = if (loadingMode) "LOADING SEQUENCE" else "UPCOMING",
                style = typography.labelSmall.copy(
                    fontWeight = FontWeight.Black,
                    fontFamily = FontFamily.Monospace,
                    letterSpacing = 1.2.sp,
                ),
                color = if (loadingMode) colorScheme.primary else colorScheme.onSurfaceVariant,
            )
            WsConnectionPill(state = wsConnectionState)
        }
        Spacer(modifier = Modifier.height(6.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                Text(
                    text = if (loadingMode) "Loading Manifest" else "Route Manifest",
                    style = typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
                    color = colorScheme.onSurface,
                )
                if (pendingCount > 0) {
                    Box(
                        modifier = Modifier
                            .size(28.dp)
                            .clip(CircleShape)
                            .background(if (loadingMode) colorScheme.primary else colorScheme.primary),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = stringResource(R.string.mobile_driver_ui_pendingcount),
                            style = typography.labelSmall.copy(fontWeight = FontWeight.Bold),
                            color = colorScheme.onPrimary,
                        )
                    }
                }
            }
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                IconButton(onClick = onRefresh) {
                    Icon(
                        Icons.Default.Refresh,
                        contentDescription = stringResource(R.string.mobile_driver_ui_refresh_manifest),
                        tint = colorScheme.onSurfaceVariant,
                    )
                }
                Text(
                    text = stringResource(R.string.mobile_driver_ui_loading_mode),
                    style = typography.labelSmall.copy(fontFamily = FontFamily.Monospace),
                    color = if (loadingMode) colorScheme.primary else colorScheme.onSurfaceVariant,
                )
                Switch(
                    checked = loadingMode,
                    onCheckedChange = { onToggleLoadingMode() }
                )
            }
        }
    }
}
