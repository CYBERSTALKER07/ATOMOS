package com.pegasusx.driver.ui.components

import android.graphics.Bitmap
import android.graphics.Canvas as AndroidCanvas
import android.graphics.Paint
import android.graphics.PorterDuff
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import java.io.ByteArrayOutputStream

/**
 * Simple ink pad for credit-leave / shop-closed PoD signatures.
 * Emits JPEG bytes when the user taps Save.
 */
@Composable
fun SignaturePad(
    onSignedJpeg: (ByteArray) -> Unit,
    modifier: Modifier = Modifier,
    heightDp: Int = 140,
) {
    var strokes by remember { mutableStateOf(listOf<List<Offset>>()) }
    var current by remember { mutableStateOf<List<Offset>>(emptyList()) }
    var canvasSize by remember { mutableStateOf(IntSize.Zero) }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(text = "Signature", fontSize = 10.sp, color = Color.Gray)
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(heightDp.dp)
                .background(Color.White)
                .border(1.dp, Color.LightGray)
                .onSizeChanged { canvasSize = it }
                .pointerInput(Unit) {
                    detectDragGestures(
                        onDragStart = { offset -> current = listOf(offset) },
                        onDrag = { change, _ ->
                            change.consume()
                            current = current + change.position
                        },
                        onDragEnd = {
                            if (current.size >= 2) {
                                strokes = strokes + listOf(current)
                            }
                            current = emptyList()
                        },
                        onDragCancel = { current = emptyList() },
                    )
                },
        ) {
            Canvas(modifier = Modifier.matchParentSize()) {
                val all = strokes + listOfNotNull(current.takeIf { it.size >= 2 })
                for (stroke in all) {
                    for (i in 1 until stroke.size) {
                        drawLine(
                            color = Color.Black,
                            start = stroke[i - 1],
                            end = stroke[i],
                            strokeWidth = 4f,
                            cap = StrokeCap.Round,
                        )
                    }
                }
            }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(
                onClick = {
                    strokes = emptyList()
                    current = emptyList()
                },
                modifier = Modifier.weight(1f),
            ) { Text("Clear") }
            OutlinedButton(
                onClick = {
                    val w = canvasSize.width.coerceAtLeast(1)
                    val h = canvasSize.height.coerceAtLeast(1)
                    val bmp = Bitmap.createBitmap(w, h, Bitmap.Config.ARGB_8888)
                    val canvas = AndroidCanvas(bmp)
                    canvas.drawColor(android.graphics.Color.WHITE, PorterDuff.Mode.SRC)
                    val paint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
                        color = android.graphics.Color.BLACK
                        style = Paint.Style.STROKE
                        strokeWidth = 4f
                        strokeCap = Paint.Cap.ROUND
                        strokeJoin = Paint.Join.ROUND
                    }
                    for (stroke in strokes) {
                        for (i in 1 until stroke.size) {
                            canvas.drawLine(
                                stroke[i - 1].x,
                                stroke[i - 1].y,
                                stroke[i].x,
                                stroke[i].y,
                                paint,
                            )
                        }
                    }
                    val out = ByteArrayOutputStream()
                    bmp.compress(Bitmap.CompressFormat.JPEG, 85, out)
                    if (!bmp.isRecycled) bmp.recycle()
                    onSignedJpeg(out.toByteArray())
                },
                modifier = Modifier.weight(1f),
                enabled = strokes.isNotEmpty(),
            ) { Text("Save signature") }
        }
    }
}
