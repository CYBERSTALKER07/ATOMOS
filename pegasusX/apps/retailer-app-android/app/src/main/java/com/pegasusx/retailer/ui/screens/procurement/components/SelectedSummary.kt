package com.pegasusx.retailer.ui.screens.procurement.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.ui.screens.procurement.ProcurementUiState
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.R

@Composable
fun SelectedSummary(selectedCount: Int, selectedUnits: Int) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                stringResource(R.string.mobile_retailer_ui_selectedcount_items_selected, selectedCount),
                modifier = Modifier.weight(1f),
                style = MaterialTheme.typography.bodyMedium,
            )
            Text(
                stringResource(R.string.mobile_retailer_ui_selectedunits_units, selectedUnits),
                style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Bold),
                color = MaterialTheme.colorScheme.primary,
            )
        }
    }
}
