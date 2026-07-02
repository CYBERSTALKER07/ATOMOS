package com.pegasusx.retailer.ui.controltower

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import com.google.android.gms.maps.model.CameraPosition
import com.google.android.gms.maps.model.LatLng
import com.google.android.gms.maps.model.MapStyleOptions
import com.google.maps.android.compose.*
import com.uber.h3core.H3Core

data class H3DensityData(
    val hex: String,
    val count: Int
)

@Composable
fun HexagonalControlTowerMap(
    data: List<H3DensityData>,
    modifier: Modifier = Modifier
) {
    val h3 = remember { H3Core.newInstance() }

    val cameraPositionState = rememberCameraPositionState {
        position = CameraPosition.fromLatLngZoom(LatLng(37.74, -122.4), 11f)
    }

    // A dark map style JSON to match the dashboard aesthetic
    val darkMapStyle = """
        [
          {
            "elementType": "geometry",
            "stylers": [
              {
                "color": "#212121"
              }
            ]
          },
          {
            "elementType": "labels.icon",
            "stylers": [
              {
                "visibility": "off"
              }
            ]
          },
          {
            "elementType": "labels.text.fill",
            "stylers": [
              {
                "color": "#757575"
              }
            ]
          },
          {
            "elementType": "labels.text.stroke",
            "stylers": [
              {
                "color": "#212121"
              }
            ]
          },
          {
            "featureType": "water",
            "elementType": "geometry",
            "stylers": [
              {
                "color": "#000000"
              }
            ]
          }
        ]
    """.trimIndent()

    GoogleMap(
        modifier = modifier.fillMaxSize(),
        cameraPositionState = cameraPositionState,
        properties = MapProperties(
            mapStyleOptions = MapStyleOptions(darkMapStyle)
        )
    ) {
        data.forEach { h3Data ->
            // Get coordinates for the hexagon boundary
            val boundary = h3.cellToBoundary(h3Data.hex).map { LatLng(it.lat, it.lng) }
            
            // Map count to an alpha value (e.g., 0 to 200 based on count out of 100 max)
            val alpha = (h3Data.count / 100f * 200).coerceIn(0f, 255f).toInt()
            
            Polygon(
                points = boundary,
                fillColor = Color(255, 255 - alpha, 0, alpha),
                strokeColor = Color.Transparent,
                strokeWidth = 0f
            )
        }
    }
}
