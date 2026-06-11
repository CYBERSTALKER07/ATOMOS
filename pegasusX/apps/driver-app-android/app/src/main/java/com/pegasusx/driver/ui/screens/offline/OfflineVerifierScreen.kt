package com.pegasusx.driver.ui.screens.offline

import android.util.Log
import android.view.ViewGroup
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.hilt.navigation.compose.hiltViewModel
import com.google.mlkit.vision.barcode.BarcodeScanning
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.common.InputImage
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed
import java.util.concurrent.Executors

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OfflineVerifierScreen(
    onBack: () -> Unit,
    viewModel: OfflineVerifierViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Offline Verifier") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(PegasusSpacing.s16),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.s12),
            ) {
                Text("Hash Manifest Protocol")
                Text("Status: ${state.statusLabel}")

                when (val verification = state.verificationState) {
                    VerificationState.Idle -> {
                        Text("Download your route manifest to begin offline cryptographic verification.")
                        Button(
                            onClick = { viewModel.syncManifest() },
                            enabled = !state.isSyncing,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(if (state.isSyncing) "Syncing…" else "Sync Route Manifest")
                        }
                    }

                    VerificationState.Syncing -> {
                        CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
                        Text("Downloading manifest…", modifier = Modifier.align(Alignment.CenterHorizontally))
                    }

                    is VerificationState.Ready -> {
                        Text("Manifest loaded — ${verification.manifest.hashes.size} orders")
                        Text("Valid: ${if (verification.manifest.isValid) "Yes" else "Expired"}")
                        state.syncedAt?.let { Text("Manifest date: $it") }
                        Button(
                            onClick = { viewModel.activateScanner() },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text("Activate Scanner")
                        }
                        OutlinedButton(
                            onClick = { viewModel.syncManifest() },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text("Re-sync Manifest")
                        }
                    }

                    VerificationState.Scanning -> Unit

                    is VerificationState.Verified -> {
                        Text("Verified order ${verification.orderId}", color = StatusGreen)
                        Button(onClick = { viewModel.nextDelivery() }, modifier = Modifier.fillMaxWidth()) {
                            Text("Next Delivery")
                        }
                    }

                    is VerificationState.Fraud -> {
                        Text(verification.reason, color = StatusRed)
                        Button(onClick = { viewModel.nextDelivery() }, modifier = Modifier.fillMaxWidth()) {
                            Text("Reset")
                        }
                    }

                    is VerificationState.Error -> {
                        Text(verification.reason, color = StatusRed)
                        state.error?.let { Text(it, color = StatusRed) }
                        Button(onClick = { viewModel.syncManifest() }, modifier = Modifier.fillMaxWidth()) {
                            Text("Retry Sync")
                        }
                    }
                }
            }

            if (state.verificationState is VerificationState.Scanning) {
                Box(modifier = Modifier.fillMaxSize()) {
                    OfflineBarcodePreview(onQrDetected = viewModel::handleBarcodeScan)
                    OutlinedButton(
                        onClick = { viewModel.cancelScanner() },
                        modifier = Modifier
                            .align(Alignment.TopCenter)
                            .padding(top = 72.dp),
                    ) {
                        Text("Cancel Scanner")
                    }
                }
            }
        }
    }
}

@Composable
private fun OfflineBarcodePreview(onQrDetected: (String) -> Unit) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    val analysisExecutor = remember { Executors.newSingleThreadExecutor() }

    DisposableEffect(Unit) {
        onDispose { analysisExecutor.shutdown() }
    }

    AndroidView(
        factory = { ctx ->
            val previewView = PreviewView(ctx).apply {
                layoutParams = ViewGroup.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.MATCH_PARENT,
                )
                scaleType = PreviewView.ScaleType.FILL_CENTER
            }

            val cameraProviderFuture = ProcessCameraProvider.getInstance(ctx)
            cameraProviderFuture.addListener({
                val cameraProvider = cameraProviderFuture.get()
                val preview = Preview.Builder().build().also {
                    it.surfaceProvider = previewView.surfaceProvider
                }
                val barcodeScanner = BarcodeScanning.getClient()

                @androidx.annotation.OptIn(androidx.camera.core.ExperimentalGetImage::class)
                val analysis = ImageAnalysis.Builder()
                    .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                    .build()
                    .also { imageAnalysis ->
                        imageAnalysis.setAnalyzer(analysisExecutor) { imageProxy ->
                            val mediaImage = imageProxy.image
                            if (mediaImage != null) {
                                val inputImage = InputImage.fromMediaImage(
                                    mediaImage,
                                    imageProxy.imageInfo.rotationDegrees,
                                )
                                barcodeScanner.process(inputImage)
                                    .addOnSuccessListener { barcodes ->
                                        for (barcode in barcodes) {
                                            if (barcode.valueType == Barcode.TYPE_TEXT ||
                                                barcode.valueType == Barcode.TYPE_URL
                                            ) {
                                                barcode.rawValue?.let(onQrDetected)
                                            }
                                        }
                                    }
                                    .addOnCompleteListener { imageProxy.close() }
                            } else {
                                imageProxy.close()
                            }
                        }
                    }

                try {
                    cameraProvider.unbindAll()
                    cameraProvider.bindToLifecycle(
                        lifecycleOwner,
                        CameraSelector.DEFAULT_BACK_CAMERA,
                        preview,
                        analysis,
                    )
                } catch (e: Exception) {
                    Log.e("OfflineVerifierScreen", "Camera bind failed", e)
                }
            }, ContextCompat.getMainExecutor(ctx))

            previewView
        },
        modifier = Modifier
            .fillMaxWidth()
            .height(360.dp),
    )
}
