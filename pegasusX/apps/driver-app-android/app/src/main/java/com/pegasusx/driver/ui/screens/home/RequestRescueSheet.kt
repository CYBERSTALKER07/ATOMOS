package com.pegasusx.driver.ui.screens.home

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.pressable
import kotlinx.coroutines.launch

private val rescueReasons = listOf(
    "Engine Failure",
    "Flat Tire",
    "Accident",
    "Other",
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RequestRescueSheet(
    api: DriverApi,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val lab = LocalPegasusColors.current
    val scope = rememberCoroutineScope()
    var reason by remember { mutableStateOf(rescueReasons.first()) }
    var note by remember { mutableStateOf("") }
    var isSubmitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    ModalBottomSheet(
        onDismissRequest = { if (!isSubmitting) onDismiss() },
        sheetState = sheetState,
        containerColor = lab.bg,
        contentColor = lab.fg,
        dragHandle = {
            Column(
                modifier = Modifier.fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Spacer(modifier = Modifier.height(12.dp))
                Spacer(
                    modifier = Modifier
                        .size(width = 32.dp, height = 4.dp)
                        .clip(RoundedCornerShape(2.dp))
                        .background(lab.fgTertiary),
                )
                Spacer(modifier = Modifier.height(16.dp))
            }
        },
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = PegasusSpacing.s20)
                .padding(bottom = 32.dp),
        ) {
            Text(
                text = "Request Rescue",
                style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
                color = lab.fg,
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = "Report a breakdown so dispatch can assign a rescue truck",
                style = MaterialTheme.typography.bodyMedium,
                color = lab.fgSecondary,
            )

            Spacer(modifier = Modifier.height(PegasusSpacing.s20))
            Text(
                text = "Rescue Reason",
                style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.SemiBold),
                color = lab.fg,
            )
            Spacer(modifier = Modifier.height(8.dp))
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                rescueReasons.forEach { option ->
                    val selected = option == reason
                    PegasusCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .pressable(onClick = { if (!isSubmitting) reason = option })
                            .then(
                                if (selected) {
                                    Modifier.border(1.dp, lab.warning, RoundedCornerShape(12.dp))
                                } else {
                                    Modifier
                                },
                            ),
                    ) {
                        Text(
                            text = option,
                            modifier = Modifier.padding(PegasusSpacing.s12),
                            style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium),
                            color = if (selected) lab.warning else lab.fg,
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(PegasusSpacing.s16))
            OutlinedTextField(
                value = note,
                onValueChange = { note = it },
                modifier = Modifier.fillMaxWidth(),
                enabled = !isSubmitting,
                label = { Text("Additional notes") },
                placeholder = { Text("Optional details…") },
                singleLine = false,
                minLines = 2,
            )

            error?.let {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodySmall,
                    color = lab.destructive,
                )
            }

            Spacer(modifier = Modifier.height(PegasusSpacing.s20))
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(lab.warning)
                    .pressable(onClick = {
                        if (isSubmitting) return@pressable
                        isSubmitting = true
                        error = null
                        scope.launch {
                            try {
                                api.requestRescue(
                                    mapOf(
                                        "reason" to reason,
                                        "note" to note.trim(),
                                    ),
                                )
                                onDismiss()
                            } catch (e: Exception) {
                                error = e.message ?: "Failed to request rescue"
                                isSubmitting = false
                            }
                        }
                    })
                    .padding(vertical = 14.dp),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (isSubmitting) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(20.dp),
                        color = lab.bg,
                        strokeWidth = 2.dp,
                    )
                } else {
                    Text(
                        text = "Request Rescue",
                        style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                        color = lab.bg,
                    )
                }
            }
        }
    }
}
