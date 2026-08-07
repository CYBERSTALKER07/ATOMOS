package com.pegasusx.retailer.ui.screens.profile.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.CreditCard
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.data.model.CreditProfile
import com.pegasusx.retailer.ui.components.RetailerMetricTile
import com.pegasusx.retailer.ui.components.RetailerStatusChip
import com.pegasusx.retailer.ui.screens.profile.CreditProfileUiState
import com.pegasusx.retailer.ui.screens.profile.CreditProfileViewModel
import com.pegasusx.retailer.ui.theme.PegasusSpacing
import java.text.NumberFormat
import java.util.Locale

@Composable
fun CreditProfileCard(
    viewModel: CreditProfileViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    CreditProfileCardContent(uiState = uiState)
}

@Composable
fun CreditProfileCardContent(
    uiState: CreditProfileUiState,
) {
    val profile = uiState.profile
    Surface(
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = Icons.Outlined.CreditCard,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = stringResource(R.string.retailer_desktop_credit_profile_card_text_supplier_credit),
                    style = MaterialTheme.typography.titleSmall,
                    modifier = Modifier.weight(1f),
                )
                if (profile != null) {
                    RetailerStatusChip(status = profile.status)
                }
            }

            when {
                uiState.isLoading -> {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(16.dp),
                            strokeWidth = 2.dp,
                        )
                        Text(
                            text = stringResource(R.string.mobile_retailer_ui_loading_credit),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
                uiState.missing -> {
                    Text(
                        text = stringResource(R.string.mobile_retailer_ui_no_credit_line_on_file_for_this_supplier_relationship),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                uiState.error != null -> {
                    Text(
                        text = uiState.error.orEmpty(),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
                profile != null -> {
                    CreditProfileMetrics(profile = profile)
                }
            }
        }
    }
}

@Composable
private fun CreditProfileMetrics(profile: CreditProfile) {
    val formatter = NumberFormat.getIntegerInstance(Locale.US)
    val util = if (profile.creditLimitMinor > 0) {
        String.format(
            Locale.US,
            "%.1f",
            (profile.currentBalanceMinor * 100.0) / profile.creditLimitMinor,
        )
    } else {
        "0.0"
    }

    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        RetailerMetricTile(
            label = stringResource(R.string.supplier_portal_credit_collections_text_limit),
            value = formatter.format(profile.creditLimitMinor),
            modifier = Modifier.weight(1f),
        )
        RetailerMetricTile(
            label = stringResource(R.string.mobile_retailer_ui_balance_due),
            value = formatter.format(profile.currentBalanceMinor),
            modifier = Modifier.weight(1f),
        )
        RetailerMetricTile(
            label = stringResource(R.string.retailer_desktop_stock_text_available),
            value = formatter.format(profile.availableCreditMinor),
            modifier = Modifier.weight(1f),
        )
    }

    Spacer(modifier = Modifier.height(4.dp))

    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        val risk = profile.riskTier.takeIf { it.isNotBlank() }
        Text(
            text = buildString {
                append("Utilization $util%")
                if (risk != null) append(" · risk $risk")
            },
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            fontFamily = FontFamily.Monospace,
            fontWeight = FontWeight.Medium,
        )
        if (profile.delinquencyCount > 0) {
            Text(
                text = stringResource(R.string.mobile_retailer_ui_delinquency_delinquencycount, profile.delinquencyCount),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.error,
            )
        }
    }
}
