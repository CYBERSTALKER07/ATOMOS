package com.pegasusx.retailer.ui.components

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
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.ui.components.RetailerTagChip
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.theme.StatusBlue
import com.pegasusx.retailer.ui.theme.StatusBlueSoft
import com.pegasusx.retailer.ui.theme.StatusGreen
import com.pegasusx.retailer.ui.theme.StatusGreenSoft
import java.util.Locale

@Composable
fun ProductCard(
    product: Product,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val isPressedState = remember { mutableStateOf(false) }
    val scale by animateFloatAsState(
        targetValue = if (isPressedState.value) 0.98f else 1f,
        animationSpec = spring(
            dampingRatio = com.pegasusx.retailer.ui.theme.MotionTokens.springFastSpatial.dampingRatio,
            stiffness = com.pegasusx.retailer.ui.theme.MotionTokens.springFastSpatial.stiffness,
        ),
        label = "product-card-scale",
    )

    Surface(
        modifier = modifier
            .fillMaxWidth()
            .scale(scale)
            .then(
                if (product.blocksAddToCart) Modifier else Modifier
            )
            .pointerInput(Unit) {
                detectTapGestures(
                    onPress = {
                        if (!product.blocksAddToCart) {
                            isPressedState.value = true
                            tryAwaitRelease()
                            isPressedState.value = false
                        }
                    },
                    onTap = { if (!product.blocksAddToCart) onClick() },
                )
            },
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Column {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(152.dp)
                    .clip(RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp))
                    .background(MaterialTheme.colorScheme.surfaceContainerHighest),
            ) {
                if (product.imageUrl != null) {
                    AsyncImage(
                        model = product.imageUrl,
                        contentDescription = product.name,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.matchParentSize(),
                    )
                } else {
                    Column(
                        modifier = Modifier.matchParentSize(),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.Center,
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.Eco,
                            contentDescription = "Product image unavailable",
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(32.dp),
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(
                            text = product.name.take(1).uppercase(),
                            style = MaterialTheme.typography.titleLarge,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }

                Row(
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .padding(12.dp)
                        .background(
                            color = MaterialTheme.colorScheme.primary,
                            shape = PillShape,
                        )
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    val listPrice = product.displayListPrice
                    if (product.hasSaleOffer && listPrice != null) {
                        Text(
                            text = listPrice,
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onPrimary.copy(alpha = 0.7f),
                            textDecoration = TextDecoration.LineThrough,
                        )
                    }
                    Text(
                        text = product.displayPrice,
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimary,
                    )
                }

                if (product.isOutOfStock) {
                    if (product.acceptsBackorder) {
                        Text(
                            text = "Backorder",
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                            modifier = Modifier
                                .align(Alignment.BottomStart)
                                .padding(12.dp)
                                .background(
                                    color = MaterialTheme.colorScheme.errorContainer,
                                    shape = PillShape,
                                )
                                .padding(horizontal = 10.dp, vertical = 6.dp),
                        )
                    } else {
                        Box(
                            modifier = Modifier
                                .matchParentSize()
                                .background(MaterialTheme.colorScheme.scrim.copy(alpha = 0.56f)),
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                text = "Out of stock",
                                style = MaterialTheme.typography.labelLarge,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onPrimary,
                            )
                        }
                    }
                } else if (product.isLowStock) {
                    Text(
                        text = "Low stock",
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onErrorContainer,
                        modifier = Modifier
                            .align(Alignment.BottomStart)
                            .padding(12.dp)
                            .background(
                                color = MaterialTheme.colorScheme.errorContainer,
                                shape = PillShape,
                            )
                            .padding(horizontal = 10.dp, vertical = 6.dp),
                    )
                }
            }

            Column(
                modifier = Modifier.padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    text = product.name,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )

                val supportingCopy = product.merchandisingLabel ?: product.description
                if (supportingCopy.isNotBlank()) {
                    Text(
                        text = supportingCopy,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }

                product.defaultVariant?.let { variant ->
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        RetailerTagChip(
                            text = variant.size,
                            bgColor = StatusBlueSoft,
                            textColor = StatusBlue,
                        )
                        if (variant.packCount > 1) {
                            RetailerTagChip(
                                text = variant.pack,
                                bgColor = StatusGreenSoft,
                                textColor = StatusGreen,
                            )
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
) = RetailerTagChip(text = text, bgColor = bgColor, textColor = textColor, modifier = modifier)
