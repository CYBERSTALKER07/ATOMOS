package com.thelab.retailer.ui.components.modifiers

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.waitForUpOrCancellation
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import com.thelab.retailer.ui.theme.MotionTokens

/**
 * A universal squish/bounce cash modifier that implements organic scaling 
 * interactions based on Material Design 3 and physical motion metaphors.
 */
fun Modifier.bounceCash(
    scaleDown: Float = 0.92f,
    onClick: () -> Unit
) = composed {
    var isPressed by remember { mutableStateOf(false) }
    val scale by animateFloatAsState(
        targetValue = if (isPressed) scaleDown else 1f,
        animationSpec = tween(
            durationMillis = if (isPressed) MotionTokens.DurationShort3 else MotionTokens.DurationMedium1,
            easing = if (isPressed) MotionTokens.EasingEmphasizedAccelerate else MotionTokens.EasingEmphasizedDecelerate
        ),
        label = "bounceScale"
    )
    
    this
        .graphicsLayer {
            scaleX = scale
            scaleY = scale
            clip = true
        }
        .pointerInput(Unit) {
            awaitPointerEventScope {
                while (true) {
                    awaitFirstDown(requireUnconsumed = false)
                    isPressed = true
                    waitForUpOrCancellation()
                    isPressed = false
                }
            }
        }
        .clickable(
            interactionSource = remember { MutableInteractionSource() },
            indication = null,
            onClick = onClick
        )
}
