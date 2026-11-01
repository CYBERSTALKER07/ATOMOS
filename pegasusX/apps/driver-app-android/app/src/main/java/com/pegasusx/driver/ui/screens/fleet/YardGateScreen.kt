package com.pegasusx.driver.ui.screens.fleet

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
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
import com.pegasusx.mobilekit.foundry.YardGateCheckEvent
import kotlinx.coroutines.launch

@Composable
fun YardGateScreen(
    driverApi: DriverApi,
    onBack: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    var depotId by remember { mutableStateOf("depot-tashkent-south") }
    var supplierId by remember { mutableStateOf("supplier-tashkent-1") }
    var vehicleId by remember { mutableStateOf("vh-isuzu-01") }
    var licensePlate by remember { mutableStateOf("01 777 AAA") }
    var direction by remember { mutableStateOf("GATE_OUT") }
    var vehiclePurpose by remember { mutableStateOf("OUTBOUND_ROUTE") }
    var boltSealBarcode by remember { mutableStateOf("SEAL-UZ-44912") }
    var sealIntact by remember { mutableStateOf(true) }

    var isSubmitting by remember { mutableStateOf(false) }
    var lastEvent by remember { mutableStateOf<YardGateCheckEvent?>(null) }
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
                text = "Yard Gate ALPR & Bolt Seal",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
            )
            OutlinedButton(onClick = onBack) {
                Text("Back")
            }
        }

        Text(
            text = "Role 9 Fleet / Security Gate · Gate-In/Out Logging & Slotting",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        // Last Event / Slotting Result
        lastEvent?.let { ev ->
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer),
            ) {
                Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text("Gate Event Recorded", fontWeight = FontWeight.Bold)
                    Text("Event ID: ${ev.gateEventId ?: "ev-${System.currentTimeMillis()}"}")
                    Text("Vehicle: ${ev.vehicleId} (${ev.licensePlate})")
                    Text("Direction: ${ev.direction} · Purpose: ${ev.vehiclePurpose}")
                    ev.assignedBaySlot?.let { bay ->
                        Box(
                            modifier = Modifier
                                .clip(RoundedCornerShape(4.dp))
                                .background(Color(0xFF1B5E20))
                                .padding(horizontal = 8.dp, vertical = 4.dp),
                        ) {
                            Text(
                                text = "ASSIGNED BAY: $bay",
                                color = Color.White,
                                fontWeight = FontWeight.Bold,
                            )
                        }
                    }
                }
            }
        }

        // Gate Direction & Purpose
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("Gate Direction", fontWeight = FontWeight.Bold)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    FilterChip(
                        selected = direction == "GATE_IN",
                        onClick = { direction = "GATE_IN" },
                        label = { Text("GATE IN") },
                    )
                    FilterChip(
                        selected = direction == "GATE_OUT",
                        onClick = { direction = "GATE_OUT" },
                        label = { Text("GATE OUT") },
                    )
                }

                Text("Vehicle Purpose", fontWeight = FontWeight.Bold)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    FilterChip(
                        selected = vehiclePurpose == "OUTBOUND_ROUTE",
                        onClick = { vehiclePurpose = "OUTBOUND_ROUTE" },
                        label = { Text("OUTBOUND") },
                    )
                    FilterChip(
                        selected = vehiclePurpose == "INBOUND_FTL",
                        onClick = { vehiclePurpose = "INBOUND_FTL" },
                        label = { Text("INBOUND") },
                    )
                    FilterChip(
                        selected = vehiclePurpose == "CROSS_DOCK",
                        onClick = { vehiclePurpose = "CROSS DOCK" },
                        label = { Text("CROSS DOCK") },
                    )
                    FilterChip(
                        selected = vehiclePurpose == "WORKSHOP",
                        onClick = { vehiclePurpose = "WORKSHOP" },
                        label = { Text("WORKSHOP") },
                    )
                }
            }
        }

        // Vehicle & Security Seal Details
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("ALPR & Physical Verification", fontWeight = FontWeight.Bold)
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        value = licensePlate,
                        onValueChange = { licensePlate = it },
                        label = { Text("License Plate (ALPR)") },
                        modifier = Modifier.weight(1f),
                    )
                    OutlinedTextField(
                        value = vehicleId,
                        onValueChange = { vehicleId = it },
                        label = { Text("Vehicle ID") },
                        modifier = Modifier.weight(1f),
                    )
                }
                OutlinedTextField(
                    value = boltSealBarcode,
                    onValueChange = { boltSealBarcode = it },
                    label = { Text("Bolt Seal Barcode") },
                    modifier = Modifier.fillMaxWidth(),
                )
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text("Seal Physically Intact & Untampered")
                    Switch(
                        checked = sealIntact,
                        onCheckedChange = { sealIntact = it },
                    )
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
                val req = YardGateCheckEvent(
                    depotId = depotId,
                    supplierId = supplierId,
                    vehicleId = vehicleId,
                    licensePlate = licensePlate,
                    direction = direction,
                    vehiclePurpose = vehiclePurpose,
                    boltSealBarcode = boltSealBarcode.ifBlank { null },
                    sealIntact = sealIntact,
                )

                scope.launch {
                    try {
                        val resp = driverApi.recordGateEvent(req)
                        isSubmitting = false
                        if (resp.isSuccessful) {
                            lastEvent = resp.body()
                        } else if (resp.code() == 409) {
                            errorMessage = "Gate Denied (409 Conflict): Outbound bolt seal invalid, broken, or vehicle Red-Tagged."
                        } else {
                            errorMessage = "Gate logging failed (HTTP ${resp.code()})"
                        }
                    } catch (e: Exception) {
                        isSubmitting = false
                        errorMessage = e.message ?: "Network error logging gate event"
                    }
                }
            },
            modifier = Modifier.fillMaxWidth(),
            enabled = !isSubmitting,
        ) {
            if (isSubmitting) {
                CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp), strokeWidth = 2.dp)
            }
            Text("Log Gate Event")
        }
    }
}
