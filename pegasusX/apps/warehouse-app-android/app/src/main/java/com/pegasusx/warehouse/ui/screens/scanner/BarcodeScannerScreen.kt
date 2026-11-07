package com.pegasusx.warehouse.ui.screens.scanner

import android.media.AudioManager
import android.media.ToneGenerator
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.platform.LocalContext
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.warehouse.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BarcodeScannerScreen(
    viewModel: ScannerViewModel = hiltViewModel(),
    onNavigateBack: () -> Unit
) {
    val state by viewModel.state.collectAsState()
    val scannedBins by viewModel.scannedBinIds.collectAsState(initial = emptyList())
    val errorMessage by viewModel.errorMessage.collectAsState()
    val context = LocalContext.current

    val toneGen = remember { ToneGenerator(AudioManager.STREAM_MUSIC, 100) }
    val vibrator = context.getSystemService(android.content.Context.VIBRATOR_SERVICE) as Vibrator

    LaunchedEffect(Unit) {
        viewModel.startScanning()
    }

    LaunchedEffect(state) {
        if (state == ScannerState.SUCCESS) {
            toneGen.startTone(ToneGenerator.TONE_PROP_BEEP, 150)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                vibrator.vibrate(VibrationEffect.createOneShot(100, VibrationEffect.DEFAULT_AMPLITUDE))
            }
        } else if (state == ScannerState.ERROR) {
            toneGen.startTone(ToneGenerator.TONE_SUP_ERROR, 300)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                vibrator.vibrate(VibrationEffect.createOneShot(300, VibrationEffect.DEFAULT_AMPLITUDE))
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Matrix Batch Scan") },
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
                    // Placeholder for ML Kit Matrix Camera Preview (Enterprise Multiple Barcodes)
                    Text("Matrix Camera Preview UI Here (Supports 10+ Barcodes Synchronously)", color = Color.White)
                }
                ScannerState.PROCESSING -> {
                    CircularProgressIndicator(color = Color.White)
                    Text(
                        stringResource(R.string.mobile_warehouse_ui_processing_lastscannedbin, scannedBins.joinToString(", ")),
                        color = Color.White,
                        modifier = Modifier.padding(top = 64.dp)
                    )
                }
                ScannerState.SUCCESS -> {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("Batch Scan Successful", color = Color.Green, style = MaterialTheme.typography.headlineMedium)
                        Text("Scanned: ${scannedBins.joinToString(", ")}", color = Color.White, modifier = Modifier.padding(16.dp))
                        Button(onClick = { viewModel.reset(); viewModel.startScanning() }) {
                            Text("Scan Next Batch")
                        }
                    }
                }
                ScannerState.ERROR -> {
                    com.pegasus.design.ui.PegasusStatePane(
                        kind = com.pegasus.design.ui.PegasusStateKind.Error,
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
