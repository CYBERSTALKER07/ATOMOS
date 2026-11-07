package com.pegasus.barcode

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.camera.core.Camera
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import com.google.mlkit.vision.barcode.BarcodeScannerOptions
import com.google.mlkit.vision.barcode.BarcodeScanning
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.common.InputImage
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicReference

/** Same-code suppress window — picker throughput (was 1500ms). */
const val EAN_SCAN_DEBOUNCE_MS = 300L

@Composable
fun EanBarcodeScannerPreview(
    onBarcode: (String) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    previewHeightDp: Int = 160,
    showTorchToggle: Boolean = true,
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    var hasCamera by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED,
        )
    }
    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted -> hasCamera = granted }

    DisposableEffect(Unit) {
        if (!hasCamera) {
            permissionLauncher.launch(Manifest.permission.CAMERA)
        }
        onDispose { }
    }

    if (!hasCamera) {
        Box(modifier.height(previewHeightDp.dp), contentAlignment = Alignment.Center) {
            Text("Camera permission required for EAN scan")
        }
        return
    }

    if (!enabled) {
        Box(
            modifier = modifier
                .fillMaxWidth()
                .height(previewHeightDp.dp),
            contentAlignment = Alignment.Center,
        ) {
            Text("Scanner paused")
        }
        return
    }

    val executor = remember { Executors.newSingleThreadExecutor() }
    var lastCode by remember { mutableStateOf("") }
    var lastEmitAt by remember { mutableLongStateOf(0L) }
    val cameraRef = remember { AtomicReference<Camera?>(null) }
    var torchOn by remember { mutableStateOf(false) }

    DisposableEffect(Unit) {
        onDispose {
            executor.shutdown()
            cameraRef.get()?.cameraControl?.enableTorch(false)
            cameraRef.set(null)
        }
    }

    Box(
        modifier = modifier
            .fillMaxWidth()
            .height(previewHeightDp.dp),
    ) {
        AndroidView(
            modifier = Modifier.matchParentSize(),
            factory = { ctx ->
                val previewView = PreviewView(ctx)
                val cameraProviderFuture = ProcessCameraProvider.getInstance(ctx)
                cameraProviderFuture.addListener({
                    val cameraProvider = cameraProviderFuture.get()
                    val preview = Preview.Builder().build().also {
                        it.surfaceProvider = previewView.surfaceProvider
                    }
                    val options = BarcodeScannerOptions.Builder()
                        .setBarcodeFormats(Barcode.FORMAT_EAN_8, Barcode.FORMAT_EAN_13)
                        .build()
                    val scanner = BarcodeScanning.getClient(options)
                    val analysis = ImageAnalysis.Builder()
                        .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                        .build()
                    analysis.setAnalyzer(executor) { imageProxy ->
                        if (!enabled) {
                            imageProxy.close()
                            return@setAnalyzer
                        }
                        val mediaImage = imageProxy.image
                        if (mediaImage == null) {
                            imageProxy.close()
                            return@setAnalyzer
                        }
                        val image = InputImage.fromMediaImage(mediaImage, imageProxy.imageInfo.rotationDegrees)
                        scanner.process(image)
                            .addOnSuccessListener { barcodes ->
                                val code = barcodes.firstOrNull()?.rawValue
                                if (!code.isNullOrBlank()) {
                                    val now = System.currentTimeMillis()
                                    val isDuplicate = code == lastCode && (now - lastEmitAt) < EAN_SCAN_DEBOUNCE_MS
                                    if (!isDuplicate) {
                                        lastCode = code
                                        lastEmitAt = now
                                        vibrateOnDetect(ctx)
                                        onBarcode(code)
                                    }
                                }
                            }
                            .addOnCompleteListener { imageProxy.close() }
                    }
                    cameraProvider.unbindAll()
                    val camera = cameraProvider.bindToLifecycle(
                        lifecycleOwner,
                        CameraSelector.DEFAULT_BACK_CAMERA,
                        preview,
                        analysis,
                    )
                    cameraRef.set(camera)
                    if (torchOn && camera.cameraInfo.hasFlashUnit()) {
                        camera.cameraControl.enableTorch(true)
                    }
                }, ContextCompat.getMainExecutor(ctx))
                previewView
            },
        )
        if (showTorchToggle) {
            FilledTonalButton(
                onClick = {
                    val cam = cameraRef.get()
                    if (cam == null || !cam.cameraInfo.hasFlashUnit()) return@FilledTonalButton
                    torchOn = !torchOn
                    cam.cameraControl.enableTorch(torchOn)
                },
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(8.dp),
            ) {
                Text(if (torchOn) "Torch on" else "Torch")
            }
        }
    }
}

/**
 * Hidden focusable field for Zebra / keyboard-wedge scanners.
 * Emits on IME Done / newline and clears the buffer.
 */
@Composable
fun KeyboardWedgeBarcodeField(
    onBarcode: (String) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    requestFocus: Boolean = true,
) {
    var buffer by remember { mutableStateOf("") }
    val focusRequester = remember { FocusRequester() }

    DisposableEffect(requestFocus, enabled) {
        if (requestFocus && enabled) {
            focusRequester.requestFocus()
        }
        onDispose { }
    }

    BasicTextField(
        value = buffer,
        onValueChange = { next ->
            if (next.contains('\n') || next.contains('\r')) {
                val code = next.replace("\r", "").replace("\n", "").trim()
                buffer = ""
                if (code.isNotEmpty()) onBarcode(code)
            } else {
                buffer = next
            }
        },
        enabled = enabled,
        singleLine = true,
        textStyle = TextStyle(color = Color.Transparent, fontSize = 1.sp),
        cursorBrush = SolidColor(Color.Transparent),
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
        keyboardActions = KeyboardActions(
            onDone = {
                val code = buffer.trim()
                buffer = ""
                if (code.isNotEmpty()) onBarcode(code)
            },
        ),
        modifier = modifier
            .fillMaxWidth()
            .height(1.dp)
            .focusRequester(focusRequester),
    )
}

private fun vibrateOnDetect(context: Context) {
    val vibrator = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
        val manager = context.getSystemService(Context.VIBRATOR_MANAGER_SERVICE) as VibratorManager
        manager.defaultVibrator
    } else {
        @Suppress("DEPRECATION")
        context.getSystemService(Context.VIBRATOR_SERVICE) as Vibrator
    }
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
        vibrator.vibrate(VibrationEffect.createOneShot(50, VibrationEffect.DEFAULT_AMPLITUDE))
    } else {
        @Suppress("DEPRECATION")
        vibrator.vibrate(50)
    }
}
