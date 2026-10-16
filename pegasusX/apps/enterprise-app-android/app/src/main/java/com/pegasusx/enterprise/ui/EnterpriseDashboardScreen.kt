package com.pegasusx.enterprise.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.*
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.automirrored.filled.Sort
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.enterprise.R
import com.pegasusx.enterprise.model.EnterpriseDashboardUiState
import com.pegasusx.enterprise.model.EnterpriseRequest
import com.pegasusx.enterprise.model.SortOption
import com.pegasusx.enterprise.theme.CADColors

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EnterpriseDashboardScreen(
    uiState: EnterpriseDashboardUiState,
    onSearchQueryChanged: (String) -> Unit,
    onCategorySelected: (String) -> Unit,
    onStatusFilterSelected: (String) -> Unit,
    onSortOptionSelected: (SortOption) -> Unit,
    onRefresh: () -> Unit,
    onInquiryClick: (EnterpriseRequest) -> Unit,
    onQuickStatusAdvance: (String, String) -> Unit,
    snackbarHostState: SnackbarHostState
) {
    val haptic = LocalHapticFeedback.current
    var isSortMenuExpanded by remember { mutableStateOf(false) }

    // Pulsing animation for live radar telemetry
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val pulseAlpha by infiniteTransition.animateFloat(
        initialValue = 0.35f,
        targetValue = 1.0f,
        animationSpec = infiniteRepeatable(
            animation = tween(1200, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulseAlpha"
    )

    Scaffold(
        containerColor = MaterialTheme.colorScheme.surface,
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            Surface(
                color = MaterialTheme.colorScheme.surfaceVariant,
                modifier = Modifier
                    .fillMaxWidth()
                    .border(width = 1.dp, color = CADColors.BorderStroke)
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 12.dp)
                ) {
                    // Header Bar with Pegasus Swift Emblem and Telemetry Status
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Box(
                                modifier = Modifier
                                    .size(48.dp)
                                    .border(1.dp, CADColors.BorderCyan, MaterialTheme.shapes.extraSmall)
                                    .background(MaterialTheme.colorScheme.surface)
                                    .padding(4.dp),
                                contentAlignment = Alignment.Center
                            ) {
                                Image(
                                    painter = painterResource(id = R.drawable.pegasus_emblem),
                                    contentDescription = "Pegasus Emblem",
                                    modifier = Modifier.fillMaxSize()
                                )
                            }
                            Spacer(modifier = Modifier.width(12.dp))
                            Column {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    Box(
                                        modifier = Modifier
                                            .size(8.dp)
                                            .background(CADColors.EmeraldGreen.copy(alpha = pulseAlpha))
                                    )
                                    Spacer(modifier = Modifier.width(8.dp))
                                    Text(
                                        text = "PEGASUS // ARCHITECTURE",
                                        style = MaterialTheme.typography.titleSmall,
                                        color = CADColors.ElectricCyan
                                    )
                                    Spacer(modifier = Modifier.width(8.dp))
                                    Text(
                                        text = "· LIVE RLS",
                                        style = MaterialTheme.typography.labelSmall,
                                        color = CADColors.TextDim
                                    )
                                }
                                Text(
                                    text = "Operator Command Console",
                                    style = MaterialTheme.typography.headlineMedium,
                                    fontWeight = FontWeight.Light,
                                    color = MaterialTheme.colorScheme.onSurface
                                )
                            }
                        }

                        // Top Controls: Sort Selector & Refresh Button (48dp Touch Targets)
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            // Sort Dropdown Menu
                            Box {
                                IconButton(
                                    onClick = {
                                        haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                        isSortMenuExpanded = true
                                    },
                                    modifier = Modifier
                                        .size(48.dp)
                                        .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                                        .background(MaterialTheme.colorScheme.surface)
                                ) {
                                    Icon(
                                        imageVector = Icons.AutoMirrored.Filled.Sort,
                                        contentDescription = "Sort Inquiries",
                                        tint = CADColors.ElectricCyan,
                                        modifier = Modifier.size(20.dp)
                                    )
                                }

                                DropdownMenu(
                                    expanded = isSortMenuExpanded,
                                    onDismissRequest = { isSortMenuExpanded = false },
                                    modifier = Modifier
                                        .background(MaterialTheme.colorScheme.surfaceVariant)
                                        .border(1.dp, CADColors.BorderStroke)
                                ) {
                                    SortOption.values().forEach { opt ->
                                        DropdownMenuItem(
                                            text = {
                                                Text(
                                                    opt.label,
                                                    style = MaterialTheme.typography.bodyMedium,
                                                    color = if (uiState.sortOption == opt) CADColors.ElectricCyan else MaterialTheme.colorScheme.onSurface
                                                )
                                            },
                                            onClick = {
                                                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                                onSortOptionSelected(opt)
                                                isSortMenuExpanded = false
                                            },
                                            leadingIcon = {
                                                if (uiState.sortOption == opt) {
                                                    Icon(Icons.Default.Check, contentDescription = null, tint = CADColors.ElectricCyan)
                                                }
                                            }
                                        )
                                    }
                                }
                            }

                            // Refresh Button
                            IconButton(
                                onClick = {
                                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                    onRefresh()
                                },
                                modifier = Modifier
                                    .size(48.dp)
                                    .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                                    .background(MaterialTheme.colorScheme.surface)
                            ) {
                                if (uiState.isRefreshing || uiState.isLoading) {
                                    CircularProgressIndicator(
                                        modifier = Modifier.size(20.dp),
                                        color = CADColors.ElectricCyan,
                                        strokeWidth = 2.dp
                                    )
                                } else {
                                    Icon(
                                        imageVector = Icons.Default.Refresh,
                                        contentDescription = "Refresh Feed",
                                        tint = MaterialTheme.colorScheme.onSurface,
                                        modifier = Modifier.size(20.dp)
                                    )
                                }
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(16.dp))

                    // OPERATIONAL KPI BENTO HUD (4 Tiles)
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        BentoKpiTile(
                            title = "QUEUE",
                            value = "${uiState.kpi.totalCount}",
                            sublabel = "Total Intake",
                            color = CADColors.ElectricCyan,
                            modifier = Modifier.weight(1f)
                        )
                        BentoKpiTile(
                            title = "ACTION",
                            value = "${uiState.kpi.newCount}",
                            sublabel = "New Leads",
                            color = CADColors.VoltLime,
                            isPulse = uiState.kpi.newCount > 0,
                            modifier = Modifier.weight(1f)
                        )
                        BentoKpiTile(
                            title = "CONVERT",
                            value = "${uiState.kpi.qualifiedCount}",
                            sublabel = "Qualified",
                            color = CADColors.EmeraldGreen,
                            modifier = Modifier.weight(1f)
                        )
                        BentoKpiTile(
                            title = "SLA",
                            value = "${uiState.kpi.avgResponseLatencyHours}h",
                            sublabel = "${uiState.kpi.dispatchHealthPercent}% Health",
                            color = CADColors.PurpleRadar,
                            modifier = Modifier.weight(1f)
                        )
                    }

                    Spacer(modifier = Modifier.height(12.dp))

                    // Material 3 Precision Search Field (56dp standard height, 4dp grid)
                    OutlinedTextField(
                        value = uiState.searchQuery,
                        onValueChange = onSearchQueryChanged,
                        placeholder = {
                            Text(
                                "Search by client name, email, company, specs, notes...",
                                color = CADColors.TextDim,
                                style = MaterialTheme.typography.bodyMedium
                            )
                        },
                        leadingIcon = {
                            Icon(
                                imageVector = Icons.Default.Search,
                                contentDescription = "Search Inquiries",
                                tint = CADColors.ElectricCyan,
                                modifier = Modifier.size(20.dp)
                            )
                        },
                        trailingIcon = {
                            if (uiState.searchQuery.isNotEmpty()) {
                                IconButton(
                                    onClick = {
                                        haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                        onSearchQueryChanged("")
                                    },
                                    modifier = Modifier.size(48.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Default.Close,
                                        contentDescription = "Clear search",
                                        tint = CADColors.TextMuted,
                                        modifier = Modifier.size(16.dp)
                                    )
                                }
                            }
                        },
                        singleLine = true,
                        shape = MaterialTheme.shapes.extraSmall,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(56.dp),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedContainerColor = MaterialTheme.colorScheme.surface,
                            unfocusedContainerColor = MaterialTheme.colorScheme.surface,
                            focusedBorderColor = CADColors.ElectricCyan,
                            unfocusedBorderColor = CADColors.BorderStroke,
                            focusedTextColor = MaterialTheme.colorScheme.onSurface,
                            unfocusedTextColor = MaterialTheme.colorScheme.onSurface
                        )
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    // Category Filter Pills (Horizontal Scroll - 4dp grid)
                    LazyRow(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        val categories = listOf(
                            Triple("all", "All Records", Icons.Outlined.Layers),
                            Triple("demo", "Demo Requests", Icons.Outlined.LocalShipping),
                            Triple("pilot", "Pilot Programs", Icons.Outlined.RocketLaunch),
                            Triple("general", "General & Inquiries", Icons.Outlined.MailOutline)
                        )
                        items(categories) { (catId, catLabel, icon) ->
                            val isSelected = uiState.selectedCategory == catId
                            FilterChip(
                                selected = isSelected,
                                onClick = {
                                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                    onCategorySelected(catId)
                                },
                                label = {
                                    Text(
                                        catLabel.uppercase(),
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Medium
                                    )
                                },
                                leadingIcon = {
                                    Icon(
                                        imageVector = icon,
                                        contentDescription = null,
                                        modifier = Modifier.size(16.dp)
                                    )
                                },
                                shape = MaterialTheme.shapes.extraSmall,
                                colors = FilterChipDefaults.filterChipColors(
                                    containerColor = MaterialTheme.colorScheme.surface,
                                    selectedContainerColor = MaterialTheme.colorScheme.onSurface,
                                    labelColor = CADColors.TextMuted,
                                    selectedLabelColor = MaterialTheme.colorScheme.surface,
                                    iconColor = CADColors.TextDim,
                                    selectedLeadingIconColor = MaterialTheme.colorScheme.surface
                                ),
                                border = FilterChipDefaults.filterChipBorder(
                                    enabled = true,
                                    selected = isSelected,
                                    borderColor = CADColors.BorderStroke,
                                    selectedBorderColor = MaterialTheme.colorScheme.onSurface,
                                    borderWidth = 1.dp
                                ),
                                modifier = Modifier.height(36.dp)
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    // Status Workflow Timeline Filter Row (4dp grid)
                    LazyRow(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        val statuses = listOf("all", "new", "in_review", "contacted", "qualified", "closed")
                        items(statuses) { st ->
                            val isSelected = uiState.selectedStatus == st
                            val stColor = if (st == "all") MaterialTheme.colorScheme.onSurface else CADColors.statusColor(st)
                            val count = if (st == "all") uiState.requests.size else uiState.requests.count { it.status.equals(st, ignoreCase = true) }

                            AssistChip(
                                onClick = {
                                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                    onStatusFilterSelected(st)
                                },
                                label = {
                                    Row(verticalAlignment = Alignment.CenterVertically) {
                                        if (st != "all") {
                                            Box(
                                                modifier = Modifier
                                                    .size(8.dp)
                                                    .background(stColor)
                                            )
                                            Spacer(modifier = Modifier.width(8.dp))
                                        }
                                        Text(
                                            text = "${st.replace("_", " ").uppercase()} ($count)",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = if (isSelected) MaterialTheme.colorScheme.surface else CADColors.TextMuted,
                                            fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal
                                        )
                                    }
                                },
                                shape = MaterialTheme.shapes.extraSmall,
                                colors = AssistChipDefaults.assistChipColors(
                                    containerColor = if (isSelected) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.surface
                                ),
                                border = AssistChipDefaults.assistChipBorder(
                                    enabled = true,
                                    borderColor = if (isSelected) MaterialTheme.colorScheme.onSurface else CADColors.BorderStroke,
                                    borderWidth = 1.dp
                                ),
                                modifier = Modifier.height(32.dp)
                            )
                        }
                    }
                }
            }
        }
    ) { paddingValues ->
        if (uiState.filteredRequests.isEmpty()) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                    modifier = Modifier.padding(24.dp)
                ) {
                    Box(
                        modifier = Modifier
                            .size(72.dp)
                            .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                            .background(MaterialTheme.colorScheme.surfaceVariant)
                            .padding(16.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        Image(
                            painter = painterResource(id = R.drawable.pegasus_emblem),
                            contentDescription = "Pegasus Emblem Watermark",
                            modifier = Modifier.fillMaxSize(),
                            alpha = 0.35f
                        )
                    }
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = "NO QUEUED INQUIRIES FOUND",
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = "No records match '${uiState.searchQuery}' under selected filter presets.",
                        style = MaterialTheme.typography.bodySmall,
                        color = CADColors.TextMuted
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    OutlinedButton(
                        onClick = {
                            haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                            onSearchQueryChanged("")
                            onCategorySelected("all")
                            onStatusFilterSelected("all")
                        },
                        shape = MaterialTheme.shapes.extraSmall,
                        colors = ButtonDefaults.outlinedButtonColors(contentColor = CADColors.ElectricCyan),
                        border = ButtonDefaults.outlinedButtonBorder(enabled = true).copy(
                            brush = Brush.horizontalGradient(listOf(CADColors.BorderCyan, CADColors.ElectricCyan))
                        ),
                        modifier = Modifier.defaultMinSize(minHeight = 48.dp)
                    ) {
                        Icon(Icons.Default.ClearAll, contentDescription = null, modifier = Modifier.size(16.dp))
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("RESET ALL FILTERS", style = MaterialTheme.typography.labelSmall)
                    }
                }
            }
        } else {
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues)
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                item {
                    // Header Status Summary Line
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                            .background(MaterialTheme.colorScheme.surfaceVariant)
                            .padding(horizontal = 12.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = "DISPLAYING ${uiState.filteredRequests.size} OF ${uiState.requests.size} SUBMISSIONS",
                            style = MaterialTheme.typography.labelSmall,
                            color = CADColors.TextDim
                        )
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                text = "ORDER: ${uiState.sortOption.label.uppercase()}",
                                style = MaterialTheme.typography.labelSmall,
                                color = CADColors.ElectricCyan
                            )
                        }
                    }
                }

                items(uiState.filteredRequests, key = { it.id }) { req ->
                    InsaneEnterpriseRequestCard(
                        request = req,
                        onClick = {
                            haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                            onInquiryClick(req)
                        },
                        onQuickAdvance = { newStatus ->
                            haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                            onQuickStatusAdvance(req.id, newStatus)
                        }
                    )
                }
            }
        }
    }
}

// Bento KPI Telemetry Metric Tile
@Composable
private fun BentoKpiTile(
    title: String,
    value: String,
    sublabel: String,
    color: Color,
    modifier: Modifier = Modifier,
    isPulse: Boolean = false
) {
    Box(
        modifier = modifier
            .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
            .background(MaterialTheme.colorScheme.surfaceContainer)
            .padding(all = 8.dp)
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.labelSmall,
                    color = CADColors.TextDim
                )
                if (isPulse) {
                    Box(
                        modifier = Modifier
                            .size(8.dp)
                            .background(color)
                    )
                }
            }
            Text(
                text = value,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                color = color
            )
            Text(
                text = sublabel,
                style = MaterialTheme.typography.labelSmall,
                color = CADColors.TextMuted,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }
    }
}

// Insane Precision CAD Request Card (M3 4dp Grid Compliant)
@Composable
private fun InsaneEnterpriseRequestCard(
    request: EnterpriseRequest,
    onClick: () -> Unit,
    onQuickAdvance: (String) -> Unit
) {
    val statusColor = CADColors.statusColor(request.status)
    val statusBg = CADColors.statusBg(request.status)

    Card(
        onClick = onClick,
        shape = MaterialTheme.shapes.extraSmall,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainer),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        border = CardDefaults.outlinedCardBorder().copy(
            brush = if (request.isUrgent) {
                Brush.horizontalGradient(listOf(CADColors.LaserOrange, CADColors.BorderStroke))
            } else {
                androidx.compose.ui.graphics.SolidColor(CADColors.BorderStroke)
            }
        ),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        ) {
            // Top Row: Avatar Monogram + Name + Verified Badge + Status Pill
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.weight(1f)
                ) {
                    // CAD Monogram Avatar Box (40dp = 4dp x 10)
                    Box(
                        modifier = Modifier
                            .size(40.dp)
                            .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                            .background(MaterialTheme.colorScheme.surface),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = request.initials,
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            color = CADColors.ElectricCyan
                        )
                    }

                    Spacer(modifier = Modifier.width(12.dp))

                    Column {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                text = request.name,
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                            if (request.status.equals("qualified", ignoreCase = true)) {
                                Spacer(modifier = Modifier.width(4.dp))
                                Icon(
                                    imageVector = Icons.Default.Verified,
                                    contentDescription = "Verified Lead",
                                    tint = CADColors.EmeraldGreen,
                                    modifier = Modifier.size(16.dp)
                                )
                            }
                        }
                        Text(
                            text = request.email,
                            style = MaterialTheme.typography.bodySmall,
                            color = CADColors.TextDim,
                            fontFamily = FontFamily.Monospace
                        )
                    }
                }

                // M3 Status Pill Badge
                Box(
                    modifier = Modifier
                        .border(1.dp, CADColors.statusBorder(request.status), MaterialTheme.shapes.extraSmall)
                        .background(statusBg)
                        .padding(horizontal = 8.dp, vertical = 4.dp)
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            modifier = Modifier
                                .size(8.dp)
                                .background(statusColor)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text(
                            text = request.status.replace("_", " ").uppercase(),
                            style = MaterialTheme.typography.labelSmall,
                            color = statusColor,
                            fontWeight = FontWeight.Bold
                        )
                    }
                }
            }

            // Company & Role Row
            if (!request.company.isNullOrBlank()) {
                Spacer(modifier = Modifier.height(8.dp))
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Outlined.Business,
                        contentDescription = "Company",
                        tint = CADColors.TextDim,
                        modifier = Modifier.size(16.dp)
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(
                        text = request.company,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.9f)
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = "•",
                        color = CADColors.TextDim,
                        style = MaterialTheme.typography.bodySmall
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = request.role ?: "Operations",
                        style = MaterialTheme.typography.bodySmall,
                        color = CADColors.TextMuted
                    )
                }
            }

            // Client Operational Message Callout Box
            if (!request.message.isNullOrBlank()) {
                Spacer(modifier = Modifier.height(8.dp))
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(MaterialTheme.colorScheme.surface)
                        .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                        .padding(horizontal = 12.dp, vertical = 8.dp)
                ) {
                    Text(
                        text = request.message,
                        style = MaterialTheme.typography.bodySmall,
                        color = CADColors.TextMuted,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // Footer Specifications Row & Quick Action Buttons
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    // Type Badge
                    Box(
                        modifier = Modifier
                            .background(MaterialTheme.colorScheme.surfaceVariant)
                            .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                            .padding(horizontal = 8.dp, vertical = 4.dp)
                    ) {
                        Text(
                            text = request.inquiry_type.uppercase(),
                            style = MaterialTheme.typography.labelSmall,
                            color = CADColors.ElectricCyan
                        )
                    }

                    // Fleet Size Tag
                    if (!request.fleet_size.isNullOrBlank()) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .background(MaterialTheme.colorScheme.surfaceVariant)
                                .border(1.dp, CADColors.BorderStroke, MaterialTheme.shapes.extraSmall)
                                .padding(horizontal = 8.dp, vertical = 4.dp)
                        ) {
                            Icon(
                                imageVector = Icons.Outlined.LocalShipping,
                                contentDescription = null,
                                tint = CADColors.TextDim,
                                modifier = Modifier.size(12.dp)
                            )
                            Spacer(modifier = Modifier.width(4.dp))
                            Text(
                                text = request.fleet_size.uppercase(),
                                style = MaterialTheme.typography.labelSmall,
                                color = CADColors.TextMuted
                            )
                        }
                    }
                }

                // Quick Inspect Button
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Text(
                        text = "REVIEW & ANSWER",
                        style = MaterialTheme.typography.labelSmall,
                        color = CADColors.ElectricCyan,
                        fontWeight = FontWeight.Bold
                    )
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                        contentDescription = "Review",
                        tint = CADColors.ElectricCyan,
                        modifier = Modifier.size(16.dp)
                    )
                }
            }
        }
    }
}
