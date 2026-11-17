package com.pegasusx.retailer.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.VerifiedUser
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.R
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.util.OwnerApprovalRequiredDetail

@Composable
fun OwnerApprovalRequiredBanner(
    detail: OwnerApprovalRequiredDetail,
    isSubmitting: Boolean = false,
    onPayCash: () -> Unit,
    onPayCard: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.secondaryContainer.copy(alpha = 0.8f),
    ) {
        Column(
            modifier = Modifier.padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.weight(1f),
                ) {
                    Icon(
                        imageVector = Icons.Outlined.VerifiedUser,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onSecondaryContainer,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = stringResource(R.string.retailer_approval_sent_title),
                        style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                }
                TextButton(onClick = onDismiss, enabled = !isSubmitting) {
                    Text("Dismiss")
                }
            }

            Text(
                text = detail.message,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSecondaryContainer.copy(alpha = 0.9f),
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    text = detail.formattedAmount,
                    style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                )
                Text(
                    text = detail.readableRuleLabel,
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSecondaryContainer.copy(alpha = 0.8f),
                )
            }

            if (detail.formattedThreshold != null || detail.formattedRemainingCap != null) {
                val capParts = buildList {
                    detail.formattedThreshold?.let { add("Limit: $it") }
                    detail.formattedRemainingCap?.let { add("Remaining: $it") }
                }
                Text(
                    text = capParts.joinToString(" · "),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSecondaryContainer.copy(alpha = 0.75f),
                )
            }

            if (detail.allowsCashFallback || detail.allowsCardFallback) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    if (detail.allowsCashFallback) {
                        Button(
                            onClick = onPayCash,
                            enabled = !isSubmitting,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(stringResource(R.string.retailer_approval_pay_cash_instead))
                        }
                    }
                    if (detail.allowsCardFallback) {
                        OutlinedButton(
                            onClick = onPayCard,
                            enabled = !isSubmitting,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(stringResource(R.string.retailer_approval_pay_card_instead))
                        }
                    }
                }
            }
        }
    }
}
