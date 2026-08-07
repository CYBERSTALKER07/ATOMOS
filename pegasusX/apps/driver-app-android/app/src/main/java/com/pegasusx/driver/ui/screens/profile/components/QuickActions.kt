package com.pegasusx.driver.ui.screens.profile.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.ShieldMoon
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.pressable

@Composable
fun QuickActions(
    onEndSession: () -> Unit = {},
    onOfflineVerifier: () -> Unit = {},
    onSyncQueue: () -> Unit = {},
    pendingCount: Int = 0,
) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        ActionRow(
            icon = Icons.Default.ShieldMoon,
            title = stringResource(R.string.mobile_driver_ui_offline_verifier),
            subtitle = "Hash manifest protocol",
            onClick = onOfflineVerifier,
        )
        ActionRow(
            icon = Icons.Default.Sync,
            title = stringResource(R.string.mobile_driver_ui_sync_queue),
            subtitle = if (pendingCount > 0) "$pendingCount pending offline actions" else "Upload pending actions",
            onClick = onSyncQueue,
        )
        ActionRow(
            icon = Icons.Default.Settings,
            title = stringResource(R.string.portal_nav_settings),
            subtitle = "App configuration",
            onClick = {}
        )
        ActionRow(
            icon = Icons.AutoMirrored.Filled.ExitToApp,
            title = stringResource(R.string.mobile_driver_ui_end_session),
            subtitle = "Go offline and sign out",
            destructive = true,
            onClick = onEndSession
        )
    }
}

@Composable
fun ActionRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    destructive: Boolean = false,
    onClick: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val tint = if (destructive) lab.destructive else lab.fg

    PegasusCard(modifier = Modifier.pressable(onClick = onClick)) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(tint.copy(alpha = 0.06f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = tint,
                    modifier = Modifier.size(15.dp)
                )
            }

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = title,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = tint
                )
                Text(
                    text = subtitle,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Medium,
                    color = lab.fgSecondary
                )
            }

            Icon(
                imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = lab.fgTertiary,
                modifier = Modifier.size(11.dp)
            )
        }
    }
}
