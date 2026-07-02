package com.pegasusx.retailer.ui.controltower

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

import com.uber.h3core.H3Core

@Composable
fun ControlTowerScreen() {
    var isNetworkView by remember { mutableStateOf(true) }
    
    // Mock Data
    val nodes = remember {
        listOf(
            NetworkNode("WH-1", "warehouse", "Central Hub"),
            NetworkNode("WH-2", "warehouse", "East DC"),
            NetworkNode("RT-1", "retailer", "Store Alpha"),
            NetworkNode("RT-2", "retailer", "Store Beta"),
            NetworkNode("DR-1", "driver", "Driver 104"),
            NetworkNode("DR-2", "driver", "Driver 211")
        )
    }

    val links = remember {
        listOf(
            NetworkLink("WH-1", "RT-1"),
            NetworkLink("WH-1", "RT-2"),
            NetworkLink("WH-2", "RT-1"),
            NetworkLink("WH-1", "DR-1"),
            NetworkLink("DR-1", "RT-1"),
            NetworkLink("WH-2", "DR-2")
        )
    }

    val h3Data = remember {
        val h3 = H3Core.newInstance()
        val data = mutableListOf<H3DensityData>()
        val centerLat = 37.74
        val centerLng = -122.4
        for (i in 0..50) {
            val lat = centerLat + (Math.random() - 0.5) * 0.5
            val lng = centerLng + (Math.random() - 0.5) * 0.5
            val hex = h3.latLngToCellAddress(lat, lng, 8)
            data.add(H3DensityData(hex, (Math.random() * 100).toInt()))
        }
        data
    }



    Box(modifier = Modifier.fillMaxSize().background(Color(0xFF0A0A0A))) {
        // Main Visualization
        if (isNetworkView) {
            LiveEKGNetworkGraph(nodes = nodes, links = links)
        } else {
            HexagonalControlTowerMap(data = h3Data)
        }

        // Header / Toggle
        Column(
            modifier = Modifier
                .padding(16.dp)
                .fillMaxWidth()
        ) {
            Text("Global Control Tower", style = MaterialTheme.typography.headlineMedium, color = Color.White)
            Text("Real-time network telematics and predictive intelligence", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
            
            Spacer(modifier = Modifier.height(16.dp))
            
            Row(
                modifier = Modifier
                    .background(Color.White.copy(alpha = 0.1f), RoundedCornerShape(8.dp))
                    .padding(4.dp)
            ) {
                TextButton(
                    onClick = { isNetworkView = true },
                    colors = ButtonDefaults.textButtonColors(
                        contentColor = if (isNetworkView) Color(0xFF34D399) else Color.Gray,
                        containerColor = if (isNetworkView) Color(0xFF34D399).copy(alpha = 0.2f) else Color.Transparent
                    )
                ) {
                    Text("Live Network Graph")
                }
                TextButton(
                    onClick = { isNetworkView = false },
                    colors = ButtonDefaults.textButtonColors(
                        contentColor = if (!isNetworkView) Color(0xFF34D399) else Color.Gray,
                        containerColor = if (!isNetworkView) Color(0xFF34D399).copy(alpha = 0.2f) else Color.Transparent
                    )
                ) {
                    Text("Spatial Map (H3)")
                }
            }
        }

        // Panels
        Surface(
            modifier = Modifier
                .align(androidx.compose.ui.Alignment.BottomStart)
                .padding(16.dp)
                .width(300.dp)
                .height(250.dp),
            color = Color.Black.copy(alpha = 0.6f),
            shape = RoundedCornerShape(12.dp)
        ) {
            Column(Modifier.padding(16.dp)) {
                Text("ACTUAL VS PLAN", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                Spacer(Modifier.height(8.dp))
                Box(modifier = Modifier.fillMaxSize().background(Color.DarkGray)) {
                    Text("Chart placeholder", modifier = Modifier.align(androidx.compose.ui.Alignment.Center))
                }
            }
        }

        Surface(
            modifier = Modifier
                .align(androidx.compose.ui.Alignment.BottomEnd)
                .padding(16.dp)
                .width(300.dp)
                .height(250.dp),
            color = Color.Black.copy(alpha = 0.6f),
            shape = RoundedCornerShape(12.dp)
        ) {
            Column(Modifier.padding(16.dp)) {
                Text("BASELINE VS UPSIDE SCENARIOS", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                Spacer(Modifier.height(8.dp))
                Box(modifier = Modifier.fillMaxSize().background(Color.DarkGray)) {
                    Text("Chart placeholder", modifier = Modifier.align(androidx.compose.ui.Alignment.Center))
                }
            }
        }
    }
}
