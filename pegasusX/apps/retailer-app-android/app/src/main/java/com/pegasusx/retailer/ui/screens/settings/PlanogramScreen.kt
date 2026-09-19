package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CameraAlt
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.PlanogramBay
import com.pegasusx.retailer.data.model.PlanogramShelf
import com.pegasusx.retailer.data.model.PlanogramSlot
import com.pegasusx.retailer.data.model.PlanogramVersion
import com.pegasusx.retailer.data.model.ShelfAudit
import com.pegasusx.retailer.data.model.ShelfAuditFinding
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class PlanogramViewModel @Inject constructor(
    val api: PegasusApi,
) : ViewModel() {
    var isPackEnabled by mutableStateOf(true)
    var activeTab by mutableStateOf(0) // 0: Shelf Slotting, 1: Aisle Walk, 2: Camera Vision
    var publishedVersion by mutableStateOf<PlanogramVersion?>(null)
    var selectedBay by mutableStateOf<PlanogramBay?>(null)
    var activeAudit by mutableStateOf<ShelfAudit?>(null)
    var bannerMessage by mutableStateOf<String?>(null)
    var isBusy by mutableStateOf(false)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlanogramScreen(
    onNavigateBack: () -> Unit,
    onNavigateToStoreStock: () -> Unit = {},
    viewModel: PlanogramViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Planograms & Shelf Vision") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            TabRow(selectedTabIndex = viewModel.activeTab) {
                Tab(
                    selected = viewModel.activeTab == 0,
                    onClick = { viewModel.activeTab = 0 },
                    text = { Text("Shelf Slotting") },
                )
                Tab(
                    selected = viewModel.activeTab == 1,
                    onClick = { viewModel.activeTab = 1 },
                    text = { Text("Aisle Walk") },
                )
                Tab(
                    selected = viewModel.activeTab == 2,
                    onClick = { viewModel.activeTab = 2 },
                    text = { Text("Camera Vision") },
                )
            }

            when (viewModel.activeTab) {
                0 -> PlanogramSlottingLayoutTab(viewModel)
                1 -> PlanogramAisleWalkTab(viewModel)
                2 -> PlanogramCameraComplianceTab(viewModel, onNavigateToStoreStock)
            }
        }
    }
}

@Composable
private fun PlanogramSlottingLayoutTab(viewModel: PlanogramViewModel) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            "Aisle 1 — Dairy & Beverage Bay A",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                        )
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = Color(0xFF2E7D32).copy(alpha = 0.15f),
                        ) {
                            Text(
                                "PUBLISHED",
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
                                fontSize = 10.sp,
                                fontWeight = FontWeight.Bold,
                                color = Color(0xFF2E7D32),
                            )
                        }
                    }
                    Spacer(Modifier.height(4.dp))
                    Text(
                        "4 Shelves · 16 Active Facings · Top-to-bottom, Left-to-right layout",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        val shelfLabels = listOf(
            "Shelf 1 (Top / Beverages)",
            "Shelf 2 (Eye Level / Milks)",
            "Shelf 3 (Yogurt & Butter)",
            "Shelf 4 (Bulk / Floor)",
        )

        items(shelfLabels) { shelfLabel ->
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text(
                        shelfLabel,
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.SemiBold,
                    )
                    Spacer(Modifier.height(8.dp))
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        repeat(4) { slotIdx ->
                            Box(
                                modifier = Modifier
                                    .width(105.dp)
                                    .height(95.dp)
                                    .clip(RoundedCornerShape(8.dp))
                                    .background(MaterialTheme.colorScheme.surface)
                                    .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(8.dp))
                                    .padding(8.dp),
                            ) {
                                Column {
                                    Text(
                                        "Slot ${slotIdx + 1}",
                                        fontSize = 10.sp,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.primary,
                                    )
                                    Spacer(Modifier.height(2.dp))
                                    Text(
                                        "SKU-${slotIdx + 101}",
                                        style = MaterialTheme.typography.bodySmall,
                                        fontWeight = FontWeight.Bold,
                                    )
                                    Spacer(Modifier.weight(1f))
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        Text(
                                            "2 facings",
                                            fontSize = 9.sp,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                        Box(
                                            modifier = Modifier
                                                .size(6.dp)
                                                .clip(RoundedCornerShape(3.dp))
                                                .background(Color(0xFF2E7D32)),
                                        )
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PlanogramAisleWalkTab(viewModel: PlanogramViewModel) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(
                        "Human Compliance Checklist",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        "Walk aisle from top to bottom, left to right. Mark any out-of-stock gaps or misplaced SKUs.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        val checklistItems = listOf(
            "Shelf 1, Slot 1 — Mineral Water 1.5L (3 facings)",
            "Shelf 1, Slot 2 — Premium Sparkling Water 1L (2 facings)",
            "Shelf 1, Slot 3 — Cola Classic 1.5L (2 facings)",
            "Shelf 2, Slot 1 — Whole Milk 3.2% (4 facings)",
            "Shelf 2, Slot 2 — Kefir 1% (2 facings)",
            "Shelf 2, Slot 3 — Greek Yogurt 5% (3 facings)",
            "Shelf 3, Slot 1 — Fruit Yogurt Berry (2 facings)",
            "Shelf 3, Slot 2 — Organic Butter 82% (2 facings)",
        )

        items(checklistItems) { itemText ->
            var checked by remember { mutableStateOf(true) }
            Card(modifier = Modifier.fillMaxWidth()) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(itemText, style = MaterialTheme.typography.bodyMedium)
                        Text(
                            if (checked) "Compliant" else "Gap Flagged",
                            color = if (checked) Color(0xFF2E7D32) else Color(0xFFC62828),
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                    Switch(checked = checked, onCheckedChange = { checked = it })
                }
            }
        }
    }
}

@Composable
private fun PlanogramCameraComplianceTab(
    viewModel: PlanogramViewModel,
    onNavigateToStoreStock: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    var isRunningInference by remember { mutableStateOf(false) }
    var auditBanner by remember { mutableStateOf<String?>(null) }

    val findings = remember {
        mutableStateListOf(
            ShelfAuditFinding("F-1", "A-1", "S-1", "GAP", "Whole Milk 3.2% (1L)", null, 0.94, "PENDING_REVIEW", 2, 1),
            ShelfAuditFinding("F-2", "A-1", "S-2", "WRONG_SKU", "Kefir 1% (500g)", "Soda Can 0.33L", 0.88, "PENDING_REVIEW", 2, 2),
            ShelfAuditFinding("F-3", "A-1", "S-4", "EMPTY", "Organic Butter 82% (200g)", null, 0.91, "PENDING_REVIEW", 3, 4),
        )
    }

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Card(
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.CameraAlt, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                        Spacer(Modifier.width(8.dp))
                        Text(
                            "Shelf Vision AI Auditor",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    Spacer(Modifier.height(6.dp))
                    Text(
                        "Snap a shelf bay photo to run dense object detection and embedding match against published planogram slots.",
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Spacer(Modifier.height(12.dp))
                    Button(
                        onClick = {
                            isRunningInference = true
                            auditBanner = null
                            scope.launch {
                                delay(1200)
                                isRunningInference = false
                                auditBanner = "Vision audit completed. 3 findings mapped to Bay A."
                            }
                        },
                        enabled = !isRunningInference,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        if (isRunningInference) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(18.dp),
                                color = MaterialTheme.colorScheme.onPrimary,
                                strokeWidth = 2.dp,
                            )
                            Spacer(Modifier.width(8.dp))
                            Text("Analyzing Shelf Image...")
                        } else {
                            Text("Capture Shelf Photo & Run Vision")
                        }
                    }
                }
            }
        }

        auditBanner?.let { banner ->
            item {
                Surface(
                    color = MaterialTheme.colorScheme.secondaryContainer,
                    shape = RoundedCornerShape(8.dp),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        banner,
                        modifier = Modifier.padding(12.dp),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                }
            }
        }

        item {
            Text(
                "Latest Vision Audit Findings (Pending Review)",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
            )
        }

        items(findings) { finding ->
            var findingStatus by remember { mutableStateOf(finding.status) }
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            imageVector = if (finding.type == "GAP") Icons.Default.Warning else Icons.Default.CheckCircle,
                            contentDescription = null,
                            tint = if (finding.type == "GAP") Color(0xFFC62828) else Color(0xFFEF6C00),
                        )
                        Spacer(Modifier.width(8.dp))
                        Text(
                            "${finding.type} — Shelf ${finding.shelfRowIndex}, Slot ${finding.slotColIndex}",
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.Bold,
                        )
                        Spacer(Modifier.weight(1f))
                        Text(
                            "${(finding.confidence * 100).toInt()}% conf",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    Spacer(Modifier.height(8.dp))
                    Text("Expected: ${finding.expectedSku}", style = MaterialTheme.typography.bodyMedium)
                    if (finding.detectedSku != null) {
                        Text("Detected: ${finding.detectedSku}", style = MaterialTheme.typography.bodySmall, color = Color(0xFFC62828))
                    }

                    Spacer(Modifier.height(12.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(
                            onClick = { findingStatus = "ACCEPTED" },
                            enabled = findingStatus == "PENDING_REVIEW",
                            modifier = Modifier.weight(1f),
                        ) {
                            Text("Accept")
                        }
                        OutlinedButton(
                            onClick = { findingStatus = "DISMISSED" },
                            enabled = findingStatus == "PENDING_REVIEW",
                            modifier = Modifier.weight(1f),
                        ) {
                            Text("Dismiss")
                        }
                    }

                    if (findingStatus == "ACCEPTED" && (finding.type == "GAP" || finding.type == "EMPTY")) {
                        Spacer(Modifier.height(8.dp))
                        TextButton(
                            onClick = onNavigateToStoreStock,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text("Create Store Stock Count Task →")
                        }
                    }
                }
            }
        }
    }
}
