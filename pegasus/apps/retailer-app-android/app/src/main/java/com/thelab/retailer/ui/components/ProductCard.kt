package com.pegasus.retailer.ui.components

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Eco
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.pegasus.retailer.data.model.Product
import com.pegasus.retailer.ui.theme.PillShape
import com.pegasus.retailer.ui.theme.SoftSquircleShape
import com.pegasus.retailer.ui.theme.StatusBlue
import com.pegasus.retailer.ui.theme.StatusBlueSoft
import com.pegasus.retailer.ui.theme.StatusGreen
import com.pegasus.retailer.ui.theme.StatusGreenSoft
import java.util.Locale

@Composable
fun ProductCard(
    product: Product,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val isPressedState = remember { mutableStateOf(false) }
    val scale by animateFloatAsState(
        if (isPressedState.value) 0.97f else 1f,
        animationSpec = spring(
            dampingRatio = com.pegasus.retailer.ui.theme.MotionTokens.springFastSpatial.dampingRatio,
            stiffness = com.pegasus.retailer.ui.theme.MotionTokens.springFastSpatial.stiffness,
        ),
        label = "press",
    )
    val shadowElevation by animateFloatAsState(if (isPressedState.value) 2f else 4f, label = "shadow")

    Surface(
        modifier = modifier
            .fillMaxWidth()
            .scale(scale)
            .shadow(
                elevation = shadowElevation.dp,
                shape = SoftSquircleShape,
                ambientColor = Color.Black.copy(alpha = 0.06f),
                spotColor = Color.Black.copy(alpha = 0.06f),
            )
            .pointerInput(Unit) {
                detectTapGestures(
                    onPress = {
                        isPressedState.value = true
                        tryAwaitRelease()
                        isPressedState.value = false
                    },
                    onTap = { onClick() },
                )
            },
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column {
            // ── Image with price overlay ──
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(140.dp)
                    .clip(RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp))
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)),
            ) {
                if (product.imageUrl != null) {
                    AsyncImage(
                        model = product.imageUrl,
                        contentDescription = product.name,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.matchParentSize(),
                    )
                } else {
                    // Placeholder
                    Column(
                        modifier = Modifier.matchParentSize(),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.Center,
                    ) {
                        androidx.compose.material3.Icon(
                            imageVector = Icons.Rounded.Eco,
                            contentDescription = null,
                            modifier = Modifier.size(28.dp),
                            tint = MaterialTheme.colorScheme.primary.copy(alpha = 0.3f),
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = product.name.take(1).uppercase(),
                            style = MaterialTheme.typography.titleLarge,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.primary.copy(alpha = 0.2f),
                        )
                    }
                }

                // Price capsule — top right
                product.defaultVariant?.let { variant ->
                    Text(
                        text = String.format(Locale.US, "$%.2f", variant.price),
                        style = MaterialTheme.typography.labelSmall.copy(
                            fontWeight = FontWeight.ExtraBold,
                            letterSpacing = 0.sp,
                        ),
                        color = Color.White,
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .padding(8.dp)
                            .background(
                                color = MaterialTheme.colorScheme.primary,
                                shape = PillShape,
                            )
                            .padding(horizontal = 10.dp, vertical = 5.dp),
                    )
                }

                // Out of stock overlay
                if (product.isOutOfStock) {
                    Box(
                        modifier = Modifier
                            .matchParentSize()
                            .background(Color.Black.copy(alpha = 0.45f)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = "Out of Stock",
                            style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                            color = Color.White,
                        )
                    }
                } else if (product.isLowStock) {
                    // Low stock badge — bottom left
                    Text(
                        text = "Low Stock",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier
                            .align(Alignment.BottomStart)
                            .padding(8.dp)
                            .background(
                                color = MaterialTheme.colorScheme.errorContainer,
                                shape = PillShape,
                            )
                            .padding(horizontal = 8.dp, vertical = 4.dp),
                    )
                }
            }

            // ── Info ──
            Column(modifier = Modifier.padding(12.dp)) {
                Text(
                    text = product.name,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )

                if (product.description.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = product.description,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }

                // Variant tag pills
                product.defaultVariant?.let { variant ->
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        TagPill(text = variant.size, bgColor = StatusBlueSoft, textColor = StatusBlue)
                        if (variant.packCount > 1) {
                            TagPill(text = variant.pack, bgColor = StatusGreenSoft, textColor = StatusGreen)
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun TagPill(
    text: String,
    bgColor: Color,
    textColor: Color,
    modifier: Modifier = Modifier,
) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelSmall.copy(
            fontSize = 10.sp,
            fontWeight = FontWeight.SemiBold,
        ),
        color = textColor,
        modifier = modifier
            .background(color = bgColor, shape = PillShape)
            .padding(horizontal = 8.dp, vertical = 3.dp),
    )
}
