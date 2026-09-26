package com.pegasusx.retailer.ui.controltower

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier

data class NetworkNode(
    val id: String = "",
    val type: String = "",
    val label: String = "",
    var x: Float = 0f,
    var y: Float = 0f
)

data class NetworkLink(
    val sourceId: String = "",
    val targetId: String = ""
)

@Composable
fun LiveEKGNetworkGraph(
    nodes: List<NetworkNode> = emptyList(),
    links: List<NetworkLink> = emptyList(),
    modifier: Modifier = Modifier,
    onNavigate: (String) -> Unit = {},
) {
    // Truthful ops pulse view
    ControlTowerScreen(onNavigate = onNavigate)
}

