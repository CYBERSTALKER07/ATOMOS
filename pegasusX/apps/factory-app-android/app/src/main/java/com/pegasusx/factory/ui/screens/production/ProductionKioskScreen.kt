package com.pegasusx.factory.ui.screens.production

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProductionKioskScreen(
    machineId: String = "MAC-123",
    onNavigateBack: () -> Unit
) {
    // This state simulates real-time WebSocket ingestion of MACHINE_JAM
    var isJammed by remember { mutableStateOf(false) }
    var currentStatus by remember { mutableStateOf("IDLE") }
    
    LaunchedEffect(isJammed) {
        if (isJammed) {
            currentStatus = "JAMMED"
        }
    }

    val bgColor = if (isJammed) Color.Red.copy(alpha = 0.8f) else MaterialTheme.colorScheme.background

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Production Kiosk - $machineId") },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = bgColor)
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .background(bgColor)
                .padding(padding)
                .padding(32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.SpaceEvenly
        ) {
            Text(
                text = "STATUS: $currentStatus",
                fontSize = 48.sp,
                fontWeight = FontWeight.Bold,
                color = if (isJammed) Color.White else MaterialTheme.colorScheme.onBackground
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly
            ) {
                Button(
                    onClick = { 
                        currentStatus = "IN_PRODUCTION"
                        isJammed = false 
                    },
                    modifier = Modifier.size(width = 250.dp, height = 150.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = Color.Green.copy(alpha = 0.7f))
                ) {
                    Text("START RUN", fontSize = 24.sp, color = Color.White)
                }

                Button(
                    onClick = { 
                        currentStatus = "PAUSED"
                        isJammed = false 
                    },
                    modifier = Modifier.size(width = 250.dp, height = 150.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = Color.DarkGray)
                ) {
                    Text("PAUSE", fontSize = 24.sp, color = Color.White)
                }
            }

            Button(
                onClick = { isJammed = !isJammed },
                modifier = Modifier.size(width = 500.dp, height = 120.dp),
                colors = ButtonDefaults.buttonColors(containerColor = Color.Red)
            ) {
                Text("FLAG QA ISSUE / JAM", fontSize = 32.sp, color = Color.White)
            }
        }
    }
}
