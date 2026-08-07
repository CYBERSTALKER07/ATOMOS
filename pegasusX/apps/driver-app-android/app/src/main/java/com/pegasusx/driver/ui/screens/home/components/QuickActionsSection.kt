package com.pegasusx.driver.ui.screens.home.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.QrCodeScanner
import androidx.compose.material.icons.filled.ShieldMoon
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.pressable

@Composable
fun QuickActionsSection(
    onScanQR: () -> Unit,
    onOfflineVerify: () -> Unit = {},
    onRequestRescue: () -> Unit = {},
    hasArrivedOrder: Boolean = false,
) {
    val lab = LocalPegasusColors.current
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(
            text = stringResource(R.string.mobile_driver_ui_quick_actions),
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            color = lab.fg,
            modifier = Modifier.padding(horizontal = PegasusSpacing.s4)
        )
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            ActionTile(
                icon = Icons.Default.QrCodeScanner,
                label = stringResource(R.string.mobile_driver_ui_scan_qr),
                modifier = Modifier.weight(1f),
                enabled = hasArrivedOrder,
                onClick = onScanQR
            )
            ActionTile(
                icon = Icons.Default.ShieldMoon,
                label = stringResource(R.string.mobile_driver_ui_offline_nverify),
                modifier = Modifier.weight(1f),
                onClick = onOfflineVerify
            )
            ActionTile(
                icon = Icons.Default.Warning,
                label = stringResource(R.string.mobile_driver_ui_rescue),
                modifier = Modifier.weight(1f),
                iconTint = lab.warning,
                onClick = onRequestRescue
            )
        }
    }
}

@Composable
private fun ActionTile(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    label: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    iconTint: Color? = null,
    onClick: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val alpha = if (enabled) 1f else 0.35f
    PegasusCard(modifier = modifier.pressable(onClick = { if (enabled) onClick() })) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(PegasusSpacing.s48)
                    .clip(CircleShape)
                    .background(lab.separator.copy(alpha = alpha)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = (iconTint ?: lab.fg).copy(alpha = alpha),
                    modifier = Modifier.size(20.dp)
                )
            }
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = lab.fgSecondary.copy(alpha = alpha),
                lineHeight = MaterialTheme.typography.labelMedium.lineHeight,
                maxLines = 2
            )
        }
    }
}
