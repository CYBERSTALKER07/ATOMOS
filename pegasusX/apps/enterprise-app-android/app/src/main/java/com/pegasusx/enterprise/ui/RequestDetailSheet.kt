package com.pegasusx.enterprise.ui

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.OpenInNew
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.enterprise.model.EnterpriseRequest
import com.pegasusx.enterprise.theme.CADColors

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RequestDetailSheet(
    request: EnterpriseRequest,
    isSendingReply: Boolean,
    onDismiss: () -> Unit,
    onStatusChange: (String) -> Unit,
    onDelete: () -> Unit,
    onSendReply: (String, String, String, String) -> Unit
) {
    val context = LocalContext.current
    val haptic = LocalHapticFeedback.current
    val scrollState = rememberScrollState()

    val firstName = remember(request.name) {
        request.name.trim().split(" ").firstOrNull() ?: request.name
    }

    var replySubject by remember(request.id) {
        mutableStateOf("Re: Pegasus Logistics ${request.inquiry_type.replaceFirstChar { it.uppercase() }} Inquiry // ${request.name}")
    }
    var replyMessage by remember(request.id) {
        mutableStateOf(
            "Hi $firstName,\n\n" +
            "Thank you for contacting the Pegasus Architecture team regarding your logistics network (${request.fleet_size ?: "fleet operations"}).\n\n" +
            "We have evaluated your submission and would like to coordinate a live platform walkthrough and sandbox provisioning for your dispatch team.\n\n" +
            "Please confirm if tomorrow at 14:00 UTC works for a 20-minute technical session.\n\n" +
            "Best regards,\n" +
            "Pegasus Enterprise Operations"
        )
    }

    var showDeleteConfirm by remember { mutableStateOf(false) }

    fun copyToClipboard(text: String, label: String) {
        val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        val clip = ClipData.newPlainText(label, text)
        clipboard.setPrimaryClip(clip)
        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
        Toast.makeText(context, "$label copied to clipboard", Toast.LENGTH_SHORT).show()
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = CADColors.PanelDark,
        shape = MaterialTheme.shapes.small,
        dragHandle = {
            Box(
                modifier = Modifier
                    .padding(vertical = 12.dp)
                    .width(44.dp)
                    .height(4.dp)
                    .background(CADColors.BorderStrong, MaterialTheme.shapes.extraSmall)
            )
        }
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp, vertical = 8.dp)
                .verticalScroll(scrollState),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Header Row: ID + Name + Status Pill
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "RECORD // ${request.id.take(8).uppercase()}",
                        style = MaterialTheme.typography.titleSmall,
                        color = CADColors.ElectricCyan
                    )
                    Text(
                        text = request.name,
                        style = MaterialTheme.typography.headlineLarge,
                        fontWeight = FontWeight.Light,
                        color = Color.White
                    )
                }

                val statusColor = CADColors.statusColor(request.status)
                Box(
                    modifier = Modifier
                        .border(1.dp, CADColors.statusBorder(request.status), MaterialTheme.shapes.extraSmall)
                        .background(CADColors.statusBg(request.status))
                        .padding(horizontal = 10.dp, vertical = 4.dp)
                ) {
                    Text(
                        text = request.status.replace("_", " ").uppercase(),
                        style = MaterialTheme.typography.labelSmall,
                        color = statusColor,
                        fontWeight = FontWeight.Bold
                    )
                }
            }

            // Interactive 5-State Visual Pipeline
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                    .background(CADColors.CardDark)
                    .padding(12.dp)
            ) {
                Text(
                    text = "WORKFLOW STATE PROGRESSION PIPELINE",
                    style = MaterialTheme.typography.labelSmall,
                    color = CADColors.TextDim
                )
                Spacer(modifier = Modifier.height(10.dp))

                val pipeline = listOf("new", "in_review", "contacted", "qualified", "closed")
                LazyRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    items(pipeline) { st ->
                        val isCurrent = request.status.equals(st, ignoreCase = true)
                        val color = CADColors.statusColor(st)

                        AssistChip(
                            onClick = {
                                if (!isCurrent) {
                                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                    onStatusChange(st)
                                }
                            },
                            label = {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    if (isCurrent) {
                                        Icon(Icons.Default.Check, contentDescription = null, modifier = Modifier.size(12.dp), tint = Color.Black)
                                        Spacer(modifier = Modifier.width(4.dp))
                                    }
                                    Text(
                                        text = st.replace("_", " ").uppercase(),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = if (isCurrent) Color.Black else color,
                                        fontWeight = if (isCurrent) FontWeight.Bold else FontWeight.Medium,
                                        fontSize = 9.sp
                                    )
                                }
                            },
                            shape = MaterialTheme.shapes.extraSmall,
                            colors = AssistChipDefaults.assistChipColors(
                                containerColor = if (isCurrent) color else CADColors.PitchBlack
                            ),
                            border = AssistChipDefaults.assistChipBorder(
                                enabled = true,
                                borderColor = if (isCurrent) color else CADColors.BorderStroke,
                                borderWidth = 1.dp
                            ),
                            modifier = Modifier.height(30.dp)
                        )
                    }
                }
            }

            // Client Operational Specifications Bento Card
            Card(
                shape = MaterialTheme.shapes.extraSmall,
                colors = CardDefaults.cardColors(containerColor = CADColors.CardDark),
                border = CardDefaults.outlinedCardBorder().copy(
                    brush = androidx.compose.ui.graphics.SolidColor(CADColors.BorderStroke)
                ),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(14.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Text(
                        text = "CLIENT SPECIFICATIONS & TELEMETRY",
                        style = MaterialTheme.typography.labelSmall,
                        color = CADColors.TextDim
                    )

                    // Email Row
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column {
                            Text("Direct Email", style = MaterialTheme.typography.bodySmall, color = CADColors.TextDim)
                            Text(
                                text = request.email,
                                style = MaterialTheme.typography.bodyMedium,
                                color = CADColors.ElectricCyan,
                                fontFamily = FontFamily.Monospace
                            )
                        }
                        IconButton(
                            onClick = { copyToClipboard(request.email, "Email") },
                            modifier = Modifier.size(36.dp)
                        ) {
                            Icon(Icons.Default.ContentCopy, contentDescription = "Copy Email", tint = CADColors.TextMuted, modifier = Modifier.size(16.dp))
                        }
                    }

                    HorizontalDivider(color = CADColors.BorderStroke, thickness = 1.dp)

                    // Company & Role Row
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column {
                            Text("Company / Organization", style = MaterialTheme.typography.bodySmall, color = CADColors.TextDim)
                            Text(request.company ?: "Independent / Unspecified", style = MaterialTheme.typography.bodyMedium, color = Color.White)
                        }
                        Column(horizontalAlignment = Alignment.End) {
                            Text("Role", style = MaterialTheme.typography.bodySmall, color = CADColors.TextDim)
                            Text(request.role ?: "Supplier Operations", style = MaterialTheme.typography.bodyMedium, color = Color.White)
                        }
                    }

                    HorizontalDivider(color = CADColors.BorderStroke, thickness = 1.dp)

                    // Inquiry Type & Fleet Scope Row
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column {
                            Text("Inquiry Type", style = MaterialTheme.typography.bodySmall, color = CADColors.TextDim)
                            Text(request.inquiry_type.uppercase(), style = MaterialTheme.typography.bodyMedium, color = CADColors.VoltLime)
                        }
                        Column(horizontalAlignment = Alignment.End) {
                            Text("Fleet Scale", style = MaterialTheme.typography.bodySmall, color = CADColors.TextDim)
                            Text(request.fleet_size ?: "Enterprise", style = MaterialTheme.typography.bodyMedium, color = Color.White)
                        }
                    }
                }
            }

            // Client Submission Message Box
            if (!request.message.isNullOrBlank()) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                        .background(CADColors.PitchBlack)
                        .padding(14.dp)
                ) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = "CLIENT SUBMISSION INQUIRY",
                            style = MaterialTheme.typography.labelSmall,
                            color = CADColors.TextDim
                        )
                        IconButton(
                            onClick = { copyToClipboard(request.message, "Client Message") },
                            modifier = Modifier.size(28.dp)
                        ) {
                            Icon(Icons.Default.ContentCopy, contentDescription = "Copy", tint = CADColors.TextDim, modifier = Modifier.size(14.dp))
                        }
                    }
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(
                        text = request.message,
                        style = MaterialTheme.typography.bodyMedium,
                        lineHeight = 20.sp,
                        color = Color.White
                    )
                }
            }

            // ANSWER CLIENT OUTBOUND STUDIO
            Card(
                shape = MaterialTheme.shapes.extraSmall,
                colors = CardDefaults.cardColors(containerColor = CADColors.CardDark),
                border = CardDefaults.outlinedCardBorder().copy(
                    brush = androidx.compose.ui.graphics.SolidColor(CADColors.BorderCyan)
                ),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(14.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = "ANSWER CLIENT // OUTBOUND RESPONSE STUDIO",
                            style = MaterialTheme.typography.titleSmall,
                            color = CADColors.VoltLime
                        )
                        Text(
                            text = "DIRECT RESEND API",
                            style = MaterialTheme.typography.labelSmall,
                            color = CADColors.TextDim,
                            fontSize = 9.sp
                        )
                    }

                    Text(
                        text = "Preset Response Blueprints:",
                        style = MaterialTheme.typography.bodySmall,
                        color = CADColors.TextMuted
                    )

                    // 4 Response Blueprints
                    LazyRow(
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        val templates = listOf(
                            Pair("Demo Booking", "Hi $firstName,\n\nWe would love to demonstrate Pegasus Logistics OS for your ${request.fleet_size ?: "network"}. Our team can host a live sandbox session on Thursday at 15:00 UTC.\n\nPlease let us know if that fits your schedule.\n\nBest,\nPegasus Team"),
                            Pair("Pilot Sandbox", "Hi $firstName,\n\nYour pilot access environment has been provisioned on Pegasus OS. We are attaching your enterprise access token and documentation.\n\nLooking forward to accelerating your fulfillment pipeline.\n\nBest,\nPegasus Engineering"),
                            Pair("Operational Specs", "Hi $firstName,\n\nTo ensure our dispatch solvers calibrate correctly for your fleet, could you share the number of primary distribution nodes and active vehicle telemetry formats?\n\nBest,\nPegasus Solutions"),
                            Pair("Executive Brief", "Hi $firstName,\n\nThank you for reaching out regarding Pegasus. I am available today for a direct briefing if you would like to discuss immediate integration timelines.\n\nWarm regards,\nPegasus Operations Lead")
                        )
                        items(templates) { (title, body) ->
                            AssistChip(
                                onClick = {
                                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                    replySubject = "Re: Pegasus ${request.inquiry_type.replaceFirstChar { it.uppercase() }} // $title"
                                    replyMessage = body
                                },
                                label = {
                                    Text(
                                        title.uppercase(),
                                        style = MaterialTheme.typography.labelSmall,
                                        fontSize = 9.sp,
                                        color = CADColors.ElectricCyan
                                    )
                                },
                                shape = MaterialTheme.shapes.extraSmall,
                                colors = AssistChipDefaults.assistChipColors(containerColor = CADColors.PitchBlack),
                                border = AssistChipDefaults.assistChipBorder(enabled = true, borderColor = CADColors.BorderStroke, borderWidth = 1.dp),
                                modifier = Modifier.height(26.dp)
                            )
                        }
                    }

                    // Subject Field
                    OutlinedTextField(
                        value = replySubject,
                        onValueChange = { replySubject = it },
                        label = { Text("Subject Line", style = MaterialTheme.typography.bodySmall) },
                        singleLine = true,
                        shape = MaterialTheme.shapes.extraSmall,
                        modifier = Modifier.fillMaxWidth(),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedContainerColor = CADColors.InputDark,
                            unfocusedContainerColor = CADColors.InputDark,
                            focusedBorderColor = CADColors.ElectricCyan,
                            unfocusedBorderColor = CADColors.BorderStroke,
                            focusedTextColor = Color.White,
                            unfocusedTextColor = Color.White
                        )
                    )

                    // Message Field with Character & Word Counter
                    OutlinedTextField(
                        value = replyMessage,
                        onValueChange = { replyMessage = it },
                        label = { Text("Client Response Body", style = MaterialTheme.typography.bodySmall) },
                        minLines = 6,
                        maxLines = 10,
                        shape = MaterialTheme.shapes.extraSmall,
                        supportingText = {
                            val words = replyMessage.trim().split("\\s+".toRegex()).filter { it.isNotEmpty() }.size
                            Text(
                                "${replyMessage.length} characters • $words words",
                                style = MaterialTheme.typography.labelSmall,
                                color = CADColors.TextDim
                            )
                        },
                        modifier = Modifier.fillMaxWidth(),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedContainerColor = CADColors.InputDark,
                            unfocusedContainerColor = CADColors.InputDark,
                            focusedBorderColor = CADColors.ElectricCyan,
                            unfocusedBorderColor = CADColors.BorderStroke,
                            focusedTextColor = Color.White,
                            unfocusedTextColor = Color.White
                        )
                    )

                    // Dual Action Buttons (48dp Touch Targets)
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        // Direct Dispatch via API
                        Button(
                            onClick = {
                                if (replyMessage.isNotBlank()) {
                                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                    onSendReply(request.id, request.email, replySubject, replyMessage)
                                }
                            },
                            enabled = !isSendingReply && replyMessage.isNotBlank(),
                            shape = MaterialTheme.shapes.extraSmall,
                            colors = ButtonDefaults.buttonColors(
                                containerColor = CADColors.ElectricCyan,
                                contentColor = Color.Black
                            ),
                            modifier = Modifier
                                .weight(1.3f)
                                .height(48.dp)
                        ) {
                            if (isSendingReply) {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(18.dp),
                                    color = Color.Black,
                                    strokeWidth = 2.dp
                                )
                            } else {
                                Icon(Icons.AutoMirrored.Filled.Send, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(modifier = Modifier.width(6.dp))
                                Text("TRANSMIT REPLY", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold)
                            }
                        }

                        // Open External Mail Client
                        OutlinedButton(
                            onClick = {
                                haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                val intent = Intent(Intent.ACTION_SENDTO).apply {
                                    data = Uri.parse("mailto:")
                                    putExtra(Intent.EXTRA_EMAIL, arrayOf(request.email))
                                    putExtra(Intent.EXTRA_SUBJECT, replySubject)
                                    putExtra(Intent.EXTRA_TEXT, replyMessage)
                                }
                                try {
                                    context.startActivity(intent)
                                } catch (e: Exception) {
                                    Toast.makeText(context, "No email client installed", Toast.LENGTH_SHORT).show()
                                }
                            },
                            shape = MaterialTheme.shapes.extraSmall,
                            colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.White),
                            border = ButtonDefaults.outlinedButtonBorder(enabled = true).copy(
                                brush = androidx.compose.ui.graphics.SolidColor(CADColors.BorderStroke)
                            ),
                            modifier = Modifier
                                .weight(1f)
                                .height(48.dp)
                        ) {
                            Icon(Icons.AutoMirrored.Filled.OpenInNew, contentDescription = null, modifier = Modifier.size(16.dp))
                            Spacer(modifier = Modifier.width(6.dp))
                            Text("MAIL APP", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }

            // Destructive Delete Action with confirmation check
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 8.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                if (!showDeleteConfirm) {
                    TextButton(
                        onClick = { showDeleteConfirm = true },
                        colors = ButtonDefaults.textButtonColors(contentColor = CADColors.LaserOrange),
                        modifier = Modifier.height(44.dp)
                    ) {
                        Icon(Icons.Default.Delete, contentDescription = null, modifier = Modifier.size(16.dp))
                        Spacer(modifier = Modifier.width(6.dp))
                        Text("DELETE INQUIRY RECORD", style = MaterialTheme.typography.labelSmall)
                    }
                } else {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text("Confirm delete?", style = MaterialTheme.typography.bodySmall, color = CADColors.LaserOrange)
                        Button(
                            onClick = {
                                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                onDelete()
                            },
                            colors = ButtonDefaults.buttonColors(containerColor = CADColors.LaserOrange, contentColor = Color.White),
                            shape = MaterialTheme.shapes.extraSmall,
                            modifier = Modifier.height(34.dp)
                        ) {
                            Text("CONFIRM", style = MaterialTheme.typography.labelSmall)
                        }
                        OutlinedButton(
                            onClick = { showDeleteConfirm = false },
                            shape = MaterialTheme.shapes.extraSmall,
                            modifier = Modifier.height(34.dp)
                        ) {
                            Text("CANCEL", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }

                TextButton(
                    onClick = onDismiss,
                    colors = ButtonDefaults.textButtonColors(contentColor = CADColors.TextMuted),
                    modifier = Modifier.height(44.dp)
                ) {
                    Text("CLOSE", style = MaterialTheme.typography.labelSmall)
                }
            }

            Spacer(modifier = Modifier.height(20.dp))
        }
    }
}
