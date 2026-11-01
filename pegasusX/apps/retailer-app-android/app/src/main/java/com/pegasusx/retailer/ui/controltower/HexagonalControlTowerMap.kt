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

private data class RenderedHexPolygon(
    val hex: String,
    val boundary: List<LatLng>,
    val fillColor: Color,
)

private val DARK_MAP_STYLE = """
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

@Composable
fun HexagonalControlTowerMap(
    data: List<H3DensityData>,
    modifier: Modifier = Modifier
) {
    val h3 = remember { H3Core.newInstance() }

    val packCenter = com.pegasus.design.sessionMapCenter()
    val centerLatLng = LatLng(packCenter?.lat ?: 0.0, packCenter?.lng ?: 0.0)

    val cameraPositionState = rememberCameraPositionState {
        position = CameraPosition.fromLatLngZoom(centerLatLng, 11f)
    }

    val mapProperties = remember {
        MapProperties(mapStyleOptions = MapStyleOptions(DARK_MAP_STYLE))
    }

    val renderedHexes = remember(data) {
        data.mapNotNull { h3Data ->
            runCatching {
                val boundary = h3.cellToBoundary(h3Data.hex).map { LatLng(it.lat, it.lng) }
                val alpha = (h3Data.count / 100f * 200).coerceIn(0f, 255f).toInt()
                RenderedHexPolygon(
                    hex = h3Data.hex,
                    boundary = boundary,
                    fillColor = Color(255, (255 - alpha).coerceIn(0, 255), 0, alpha),
                )
            }.getOrNull()
        }
    }

    GoogleMap(
        modifier = modifier.fillMaxSize(),
        cameraPositionState = cameraPositionState,
        properties = mapProperties,
    ) {
        renderedHexes.forEach { hexPolygon ->
            key(hexPolygon.hex) {
                Polygon(
                    points = hexPolygon.boundary,
                    fillColor = hexPolygon.fillColor,
                    strokeColor = Color(0x44F59E0B),
                    strokeWidth = 2f
                )
            }
        }
    }
}
