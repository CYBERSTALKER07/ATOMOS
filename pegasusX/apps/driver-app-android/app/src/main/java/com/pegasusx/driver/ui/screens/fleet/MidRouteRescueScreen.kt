package com.pegasusx.driver.ui.screens.fleet

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.mobilekit.foundry.MidRouteRescueRequest
import com.pegasusx.mobilekit.foundry.MidRouteRescueResult
import com.pegasusx.mobilekit.foundry.RescueToteItem
import kotlinx.coroutines.launch

@Composable
fun MidRouteRescueScreen(
    driverApi: DriverApi,
    onBack: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    var originalRouteId by remember { mutableStateOf("route-tashkent-042") }
    var brokenVehicleId by remember { mutableStateOf("vh-isuzu-01") }
    var originalDriverId by remember { mutableStateOf("driver-uz-09") }
    var rescueVehicleId by remember { mutableStateOf("vh-isuzu-02") }
    var rescueDriverId by remember { mutableStateOf("driver-uz-11") }
    var lastCompletedStopSeq by remember { mutableStateOf("3") }
    var newRescueBoltSeal by remember { mutableStateOf("SEAL-RESCUE-8841") }

    var totes by remember {
        mutableStateOf(
            listOf(
                RescueToteItem("tote-01", 1, false, 45000000L, 10, 30),
                RescueToteItem("tote-02", 2, false, 80000000L, 15, 30),
                RescueToteItem("tote-03", 3, false, 65000000L, 20, 30),
                RescueToteItem("tote-04", 4, true, 120000000L, 45, 30), // Exceeded max excursion!
                RescueToteItem("tote-05", 5, true, 95000000L, 25, 30),
                RescueToteItem("tote-06", 6, false, 50000000L, 0, 30),
            )
        )
    }

    var isSubmitting by remember { mutableStateOf(false) }
    var result by remember { mutableStateOf<MidRouteRescueResult?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(scrollState)
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.s16),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "Mid-Route Breakdown Rescue",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
            )
            OutlinedButton(onClick = onBack) {
                Text("Back")
            }
        }

        Text(
            text = "Role 9 Fleet Safety · Dynamic Driver Liability Split & Thermal Triage",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        // Result Card
        result?.let { res ->
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer),
            ) {
                Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Rescue Executed Successfully", fontWeight = FontWeight.Bold)
                    Text("Rescue Route: ${res.rescueRouteId}")
                    Text("Original Driver Liability: ${res.originalDriverLiabilityMinor / 100} UZS (${res.originalDriverLiabilityMinor} Tiyin)")
                    Text("Rescue Driver Liability: ${res.rescueDriverLiabilityMinor / 100} UZS (${res.rescueDriverLiabilityMinor} Tiyin)")
                    Text("Transferred Totes (${res.transferredToteIds.size}): ${res.transferredToteIds.joinToString(", ")}")
                    if (res.quarantinedSpoiledToteIds.isNotEmpty()) {
                        Text(
                            text = "QUARANTINED SPOILED (${res.quarantinedSpoiledToteIds.size}): ${res.quarantinedSpoiledToteIds.joinToString(", ")} (MARKING:THERMAL_QUARANTINE_SPOILED)",
                            color = MaterialTheme.colorScheme.error,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    Text("New Bolt Seal: ${res.newRescueBoltSeal}")
                }
            }
        }

        // Vehicles & Drivers Configuration
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("Rescue Transfer Parameters", fontWeight = FontWeight.Bold)
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        value = brokenVehicleId,
                        onValueChange = { brokenVehicleId = it },
                        label = { Text("Broken Vehicle") },
                        modifier = Modifier.weight(1f),
                    )
                    OutlinedTextField(
                        value = originalDriverId,
                        onValueChange = { originalDriverId = it },
                        label = { Text("Original Driver") },
                        modifier = Modifier.weight(1f),
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        value = rescueVehicleId,
                        onValueChange = { rescueVehicleId = it },
                        label = { Text("Rescue Vehicle") },
                        modifier = Modifier.weight(1f),
                    )
                    OutlinedTextField(
                        value = rescueDriverId,
                        onValueChange = { rescueDriverId = it },
                        label = { Text("Rescue Driver") },
                        modifier = Modifier.weight(1f),
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        value = lastCompletedStopSeq,
                        onValueChange = { lastCompletedStopSeq = it },
                        label = { Text("Last Stop Seq (K)") },
                        modifier = Modifier.weight(1f),
                    )
                    OutlinedTextField(
                        value = newRescueBoltSeal,
                        onValueChange = { newRescueBoltSeal = it },
                        label = { Text("New Bolt Seal") },
                        modifier = Modifier.weight(1f),
                    )
                }
            }
        }

        // Tote Inventory & Cold Chain Triage
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Cargo Totes & Thermal Triage", fontWeight = FontWeight.Bold)
                Text(
                    text = "Totes past Stop $lastCompletedStopSeq will transfer. Excursion > Max triggers thermal quarantine.",
                    style = MaterialTheme.typography.bodySmall,
                )
                HorizontalDivider()

                val stopK = lastCompletedStopSeq.toLongOrNull() ?: 0L
                totes.forEach { tote ->
                    val isCompleted = tote.stopSequence <= stopK
                    val isSpoiled = tote.isColdChain && (tote.thermalExcursionMinutes > tote.maxAllowedExcursionMins)

                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Column {
                            Text(
                                text = "${tote.toteId} · Stop #${tote.stopSequence} · ${tote.expectedCashMinor / 100} UZS",
                                fontWeight = FontWeight.SemiBold,
                            )
                            if (tote.isColdChain) {
                                Text(
                                    text = "Cold Chain: ${tote.thermalExcursionMinutes}m / ${tote.maxAllowedExcursionMins}m limit",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = if (isSpoiled) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }

                        if (isCompleted) {
                            Text("Locked (Driver 1)", color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.labelSmall)
                        } else if (isSpoiled) {
                            Box(
                                modifier = Modifier
                                    .clip(RoundedCornerShape(4.dp))
                                    .background(Color(0xFFFFEBEE))
                                    .padding(horizontal = 6.dp, vertical = 2.dp),
                            ) {
                                Text("SPOILED (HOLD)", color = Color(0xFFC62828), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelSmall)
                            }
                        } else {
                            Text("Transfer → Driver 2", color = Color(0xFF2E7D32), style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }
        }

        errorMessage?.let {
            Text(it, color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.SemiBold)
        }

        Button(
            onClick = {
                isSubmitting = true
                errorMessage = null
                val k = lastCompletedStopSeq.toLongOrNull() ?: 0L
                val req = MidRouteRescueRequest(
                    cellId = "cell-uz",
                    supplierId = "supplier-tashkent-1",
                    originalRouteId = originalRouteId,
                    brokenVehicleId = brokenVehicleId,
                    originalDriverId = originalDriverId,
                    rescueVehicleId = rescueVehicleId,
                    rescueDriverId = rescueDriverId,
                    lastCompletedStopSeq = k,
                    newRescueBoltSeal = newRescueBoltSeal,
                    totes = totes,
                )

                scope.launch {
                    try {
                        val resp = driverApi.executeMidRouteRescue(req)
                        isSubmitting = false
                        if (resp.isSuccessful) {
                            result = resp.body()
                        } else {
                            errorMessage = "Rescue execution failed (HTTP ${resp.code()})"
                        }
                    } catch (e: Exception) {
                        isSubmitting = false
                        errorMessage = e.message ?: "Network error during rescue execution"
                    }
                }
            },
            modifier = Modifier.fillMaxWidth(),
            enabled = !isSubmitting,
        ) {
            if (isSubmitting) {
                CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp), strokeWidth = 2.dp)
            }
            Text("Execute Rescue Handover")
        }
    }
}
