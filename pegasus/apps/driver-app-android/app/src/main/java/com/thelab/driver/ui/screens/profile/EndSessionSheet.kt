package com.thelab.driver.ui.screens.profile

import androidx.compose.animation.AnimatedVisibility
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Build
import androidx.compose.material.icons.filled.NightsStay
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.QuestionMark
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.thelab.driver.ui.components.LabCard
import com.thelab.driver.ui.theme.LabSpacing
import com.thelab.driver.ui.theme.LocalLabColors
import com.thelab.driver.ui.theme.pressable

enum class OfflineReason(val code: String, val label: String, val icon: ImageVector) {
    SHIFT_COMPLETE("SHIFT_COMPLETE", "Shift Complete", Icons.Default.NightsStay),
    TRUCK_DAMAGED("TRUCK_DAMAGED", "Truck Damaged", Icons.Default.Build),
    PERSONAL("PERSONAL", "Personal", Icons.Default.Person),
    OTHER("OTHER", "Other", Icons.Default.QuestionMark)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EndSessionSheet(
    hasActiveOrders: Boolean,
    isEnding: Boolean,
    error: String?,
    onEndSession: (reason: String, note: String?) -> Unit,
    onDismiss: () -> Unit
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val lab = LocalLabColors.current
    var selectedReason by remember { mutableStateOf<OfflineReason?>(null) }
    var note by remember { mutableStateOf("") }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = lab.bg,
        contentColor = lab.fg,
        dragHandle = {
            Column(
                modifier = Modifier.fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Spacer(modifier = Modifier.height(12.dp))
                Spacer(
                    modifier = Modifier
                        .size(width = 32.dp, height = 4.dp)
                        .clip(RoundedCornerShape(2.dp))
                        .background(lab.fgTertiary)
                )
                Spacer(modifier = Modifier.height(16.dp))
            }
        }
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = LabSpacing.s20)
                .padding(bottom = 32.dp)
        ) {
            // Title
            Text(
                text = "End Session",
                style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
                color = lab.fg
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = "Go offline and end your driving session",
                style = MaterialTheme.typography.bodyMedium,
                color = lab.fgSecondary
            )

            // Active orders warning
            if (hasActiveOrders) {
                Spacer(modifier = Modifier.height(LabSpacing.s16))
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(12.dp))
                        .background(lab.destructive.copy(alpha = 0.08f))
                        .padding(LabSpacing.s12),
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.Warning,
                        contentDescription = null,
                        tint = lab.destructive,
                        modifier = Modifier.size(18.dp)
                    )
                    Text(
                        text = "You have active orders. Complete or return them before ending your session.",
                        style = MaterialTheme.typography.bodySmall,
                        color = lab.destructive
                    )
                }
            }

            Spacer(modifier = Modifier.height(LabSpacing.s20))

            // Reason selection
            Text(
                text = "REASON",
                style = MaterialTheme.typography.labelSmall.copy(
                    fontWeight = FontWeight.Black,
                    letterSpacing = 1.sp
                ),
                color = lab.fgTertiary
            )
            Spacer(modifier = Modifier.height(LabSpacing.s8))

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OfflineReason.entries.forEach { reason ->
                    ReasonOption(
                        reason = reason,
                        isSelected = selectedReason == reason,
                        onClick = { selectedReason = reason }
                    )
                }
            }

            // Note field — required for OTHER, optional for TRUCK_DAMAGED
            AnimatedVisibility(
                visible = selectedReason == OfflineReason.OTHER || selectedReason == OfflineReason.TRUCK_DAMAGED
            ) {
                Column {
                    Spacer(modifier = Modifier.height(LabSpacing.s16))
                    OutlinedTextField(
                        value = note,
                        onValueChange = { note = it },
                        label = {
                            Text(
                                if (selectedReason == OfflineReason.OTHER) "Describe reason (required)"
                                else "Describe damage (optional)"
                            )
                        },
                        modifier = Modifier.fillMaxWidth(),
                        minLines = 2,
                        maxLines = 4
                    )
                }
            }

            // Error
            if (error != null) {
                Spacer(modifier = Modifier.height(LabSpacing.s8))
                Text(
                    text = error,
                    style = MaterialTheme.typography.bodySmall,
                    color = lab.destructive
                )
            }

            Spacer(modifier = Modifier.height(LabSpacing.s24))

            // Confirm button
            val canConfirm = selectedReason != null &&
                !hasActiveOrders &&
                !isEnding &&
                (selectedReason != OfflineReason.OTHER || note.isNotBlank())

            LabCard(
                modifier = Modifier
                    .fillMaxWidth()
                    .pressable(enabled = canConfirm) {
                        selectedReason?.let { reason ->
                            onEndSession(
                                reason.code,
                                note.ifBlank { null }
                            )
                        }
                    }
                    .then(
                        if (canConfirm) Modifier.background(
                            lab.destructive,
                            RoundedCornerShape(16.dp)
                        )
                        else Modifier.background(
                            lab.fgTertiary.copy(alpha = 0.3f),
                            RoundedCornerShape(16.dp)
                        )
                    )
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 14.dp),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    if (isEnding) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(18.dp),
                            strokeWidth = 2.dp,
                            color = lab.buttonFg
                        )
                    } else {
                        Text(
                            text = "End Session",
                            style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                            color = if (canConfirm) lab.buttonFg else lab.fgTertiary
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ReasonOption(
    reason: OfflineReason,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    val lab = LocalLabColors.current

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .then(
                if (isSelected) Modifier.border(1.5.dp, lab.fg, RoundedCornerShape(12.dp))
                else Modifier.border(1.dp, lab.fgTertiary.copy(alpha = 0.2f), RoundedCornerShape(12.dp))
            )
            .background(if (isSelected) lab.fg.copy(alpha = 0.04f) else lab.bg)
            .pressable(onClick = onClick)
            .padding(LabSpacing.s16),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(14.dp)
    ) {
        Icon(
            imageVector = reason.icon,
            contentDescription = null,
            tint = if (isSelected) lab.fg else lab.fgSecondary,
            modifier = Modifier.size(20.dp)
        )
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = reason.label,
                style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.SemiBold),
                color = if (isSelected) lab.fg else lab.fgSecondary
            )
        }
    }
}
