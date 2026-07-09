package com.pegasusx.warehouse.ui.screens.scanner

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BarcodeScannerScreen(
    viewModel: ScannerViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit
) {
    val state by viewModel.state.collectAsState()
    val lastScannedBin by viewModel.lastScannedBinId.collectAsState()
    val errorMessage by viewModel.errorMessage.collectAsState()

    LaunchedEffect(Unit) {
        viewModel.startScanning()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Scan Bin") },
                navigationIcon = {
                    // Back button icon would go here
                }
            )
        }
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .background(Color.Black),
            contentAlignment = Alignment.Center
        ) {
            when (state) {
                ScannerState.SCANNING -> {
                    // Placeholder for Camera Preview
                    Text("Camera Preview UI Here", color = Color.White)
                }
                ScannerState.PROCESSING -> {
                    CircularProgressIndicator(color = Color.White)
                    Text(
                        "Processing $lastScannedBin...",
                        color = Color.White,
                        modifier = Modifier.padding(top = 64.dp)
                    )
                }
                ScannerState.SUCCESS -> {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("Successfully Scanned", color = Color.Green, style = MaterialTheme.typography.headlineMedium)
                        Text("Bin: $lastScannedBin", color = Color.White, modifier = Modifier.padding(16.dp))
                        Button(onClick = { viewModel.reset(); viewModel.startScanning() }) {
                            Text("Scan Another Bin")
                        }
                    }
                }
                ScannerState.ERROR -> {
                    com.pegasus.design.PegasusStatePane(
                        kind = com.pegasus.design.PegasusStateKind.Error,
                        headline = "Scanner Error",
                        body = errorMessage ?: "Unknown error",
                        actionLabel = "Try Again",
                        onAction = { viewModel.reset(); viewModel.startScanning() }
                    )
                }
                ScannerState.IDLE -> {
                    // Wait for start
                }
            }
        }
    }
}
