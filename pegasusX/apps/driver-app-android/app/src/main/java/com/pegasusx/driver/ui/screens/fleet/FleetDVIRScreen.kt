package com.pegasusx.driver.ui.screens.fleet

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import com.pegasusx.driver.ui.theme.PegasusSpacing

@Composable
fun FleetDVIRScreen(
    viewModel: DVIRViewModel,
    onNavigateToRescue: () -> Unit = {},
    onNavigateToGate: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val state by viewModel.uiState.collectAsState()
    val scrollState = rememberScrollState()

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(scrollState)
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.s16),
    ) {
        Text(
            text = "Pre-Trip DVIR Walkaround",
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
        )
        Text(
            text = "Role 9 Fleet Safety Gate · 8-Point Physical Inspection",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        // Red-Tag Grounding Banner
        if (state.isRedTagged) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(8.dp))
                    .background(Color(0xFFB00020))
                    .padding(16.dp),
            ) {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Icon(
                            imageVector = Icons.Default.Warning,
                            contentDescription = "Red Tag Warning",
                            tint = Color.White,
                        )
                        Text(
                            text = "VEHICLE RED-TAGGED: OUT OF SERVICE",
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                            fontSize = 16.sp,
                        )
                    }
                    Text(
                        text = "Status: RED_TAG_QUARANTINE. Critical defect detected. Driver pairing severed and vehicle locked from route departure.",
                        color = Color.White,
                        style = MaterialTheme.typography.bodySmall,
                    )
                    state.redTagWorkOrderId?.let { woId ->
                        Text(
                            text = "Maintenance Work Order: $woId",
                            color = Color.White,
                            fontWeight = FontWeight.SemiBold,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                    if (state.redTagFailedPoints.isNotEmpty()) {
                        Text(
                            text = "Defects: ${state.redTagFailedPoints.joinToString(", ")}",
                            color = Color.White,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }
            }
        }

        // Passed Clearance Badge
        if (state.dispatchAuthorized == true && !state.isRedTagged) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(8.dp))
                    .background(Color(0xFF1B5E20))
                    .padding(16.dp),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.CheckCircle,
                        contentDescription = "Cleared",
                        tint = Color.White,
                    )
                    Column {
                        Text(
                            text = "DISPATCH AUTHORIZED",
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                        )
                        Text(
                            text = "8-Point DVIR verification passed. Vehicle cleared for outbound loading.",
                            color = Color.White,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }
            }
        }

        // Vehicle & Odometer Info
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("Inspection Parameters", fontWeight = FontWeight.Bold)
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        value = state.vehicleId,
                        onValueChange = { viewModel.updateVehicleId(it) },
                        label = { Text("Vehicle ID") },
                        modifier = Modifier.weight(1f),
                        singleLine = true,
                    )
                    OutlinedTextField(
                        value = state.driverId,
                        onValueChange = { viewModel.updateDriverId(it) },
                        label = { Text("Driver ID") },
                        modifier = Modifier.weight(1f),
                        singleLine = true,
                    )
                }
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    OutlinedTextField(
                        value = state.odometerKm.toString(),
                        onValueChange = {
                            val parsed = it.toLongOrNull()
                            if (parsed != null) viewModel.updateOdometerKm(parsed)
                        },
                        label = { Text("Odometer (km)") },
                        modifier = Modifier.weight(1f),
                        singleLine = true,
                    )
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("Cold Chain")
                        Spacer(Modifier.width(8.dp))
                        Switch(
                            checked = state.requiresColdChain,
                            onCheckedChange = { viewModel.updateRequiresColdChain(it) },
                        )
                    }
                }
                if (state.requiresColdChain) {
                    OutlinedTextField(
                        value = state.reeferTempCelsius.toString(),
                        onValueChange = {
                            val parsed = it.toDoubleOrNull()
                            if (parsed != null) viewModel.updateReeferTempCelsius(parsed)
                        },
                        label = { Text("Reefer Temp (°C)") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                }
            }
        }

        // 8-Point Checklist Items
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("8 Safety Walkaround Points", fontWeight = FontWeight.Bold)
                HorizontalDivider()

                DVIRCheckRow(
                    label = "1. Steering Mechanism & Horn",
                    checked = state.checklist.steeringAndHornOk,
                    onCheckedChange = { ok ->
                        viewModel.updateChecklist { it.copy(steeringAndHornOk = ok) }
                    },
                )
                DVIRCheckRow(
                    label = "2. Service & Parking Brakes",
                    checked = state.checklist.serviceAndParkingBrakesOk,
                    onCheckedChange = { ok ->
                        viewModel.updateChecklist { it.copy(serviceAndParkingBrakesOk = ok) }
                    },
                )
                DVIRCheckRow(
                    label = "3. Tires Tread & Pressure",
                    checked = state.checklist.tiresTreadAndPressureOk,
                    onCheckedChange = { ok ->
                        viewModel.updateChecklist { it.copy(tiresTreadAndPressureOk = ok) }
                    },
                )
                DVIRCheckRow(
                    label = "4. Lights, Indicators & Hazards",
                    checked = state.checklist.lightsAndHazardsOk,
                    onCheckedChange = { ok ->
                        viewModel.updateChecklist { it.copy(lightsAndHazardsOk = ok) }
                    },
                )
                DVIRCheckRow(
                    label = "5. Wipers & Fluid Levels",
                    checked = state.checklist.wipersAndFluidsOk,
                    onCheckedChange = { ok ->
                        viewModel.updateChecklist { it.copy(wipersAndFluidsOk = ok) }
                    },
                )
                DVIRCheckRow(
                    label = "6. Emergency Kit & Fire Extinguisher",
                    checked = state.checklist.emergencyKitAndExtinguisherOk,
                    onCheckedChange = { ok ->
                        viewModel.updateChecklist { it.copy(emergencyKitAndExtinguisherOk = ok) }
                    },
                )
                DVIRCheckRow(
                    label = "7. Cargo Box Sanitation & Food Safety",
                    checked = state.checklist.cargoBoxSanitationOk,
                    onCheckedChange = { ok ->
                        viewModel.updateChecklist { it.copy(cargoBoxSanitationOk = ok) }
                    },
                )
                if (state.requiresColdChain) {
                    DVIRCheckRow(
                        label = "8. Reefer Temperature Calibration",
                        checked = state.checklist.reeferTempCalibratedOk,
                        onCheckedChange = { ok ->
                            viewModel.updateChecklist { it.copy(reeferTempCalibratedOk = ok) }
                        },
                    )
                }
            }
        }

        // Status or Error Messages
        state.statusMessage?.let {
            Text(it, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
        }
        state.errorMessage?.let {
            Text(it, color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.SemiBold)
        }

        // Actions
        val haptic = LocalHapticFeedback.current
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Button(
                onClick = {
                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                    viewModel.submitDVIR()
                },
                modifier = Modifier.weight(1f).height(48.dp),
                enabled = !state.isSubmitting,
            ) {
                if (state.isSubmitting) {
                    CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp), strokeWidth = 2.dp)
                }
                Text("Submit Inspection", fontWeight = FontWeight.Bold)
            }
            OutlinedButton(
                onClick = {
                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                    viewModel.checkDispatchAuthorization()
                },
                modifier = Modifier.height(48.dp),
                enabled = !state.isSubmitting,
            ) {
                Text("Verify Gate")
            }
        }

        // Quick navigation shortcuts to rescue and gate
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            OutlinedButton(
                onClick = {
                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                    onNavigateToRescue()
                },
                modifier = Modifier.weight(1f).height(48.dp),
            ) {
                Text("Mid-Route Rescue")
            }
            OutlinedButton(
                onClick = {
                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                    onNavigateToGate()
                },
                modifier = Modifier.weight(1f).height(48.dp),
            ) {
                Text("Yard Gate Check")
            }
        }
    }
}

@Composable
private fun DVIRCheckRow(
    label: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    val haptic = LocalHapticFeedback.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = if (!checked) FontWeight.Bold else FontWeight.Normal,
            color = if (!checked) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurface,
        )
        Switch(
            checked = checked,
            onCheckedChange = {
                haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                onCheckedChange(it)
            },
        )
    }
}
