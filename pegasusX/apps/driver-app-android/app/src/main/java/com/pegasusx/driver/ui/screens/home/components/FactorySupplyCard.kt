package com.pegasusx.driver.ui.screens.home.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.pressable
import com.pegasusx.driver.R

@Composable
fun FactorySupplyCard(onOpenSupplyTransfers: () -> Unit) {
    val lab = LocalPegasusColors.current
    val factoryLabel = TokenHolder.factoryName?.takeIf { it.isNotBlank() } ?: "Factory depot"

    PegasusCard(modifier = Modifier.pressable(onClick = onOpenSupplyTransfers)) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Icon(
                imageVector = Icons.Default.LocalShipping,
                contentDescription = null,
                tint = lab.fg,
                modifier = Modifier.size(28.dp),
            )
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = stringResource(R.string.mobile_driver_ui_supply_transfers),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Bold,
                    color = lab.fg,
                )
                Text(
                    text = stringResource(R.string.mobile_driver_ui_factorylabel_warehouse_legs),
                    style = MaterialTheme.typography.bodySmall,
                    color = lab.fgTertiary,
                )
            }
            Icon(
                imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                contentDescription = null,
                tint = lab.fgTertiary,
            )
        }
    }
}
