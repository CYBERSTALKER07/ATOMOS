package com.pegasusx.supplier.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.pegasusx.supplier.data.model.DispatchProposedRoute
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style
import org.maplibre.android.style.layers.LineLayer
import org.maplibre.android.style.layers.Property
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.LineString
import org.maplibre.geojson.Point
import com.pegasusx.supplier.R

private const val STYLE_URL = "https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
private val ROUTE_COLORS = listOf(
    "#1b6ef3",
    "#0f9d58",
    "#db4437",
    "#f4b400",
    "#ab47bc",
    "#00838f",
)

@Composable
fun DispatchPreviewMapLibre(
    routes: List<DispatchProposedRoute>,
    modifier: Modifier = Modifier,
) {
    val routable = routes.filter { route ->
        route.routeGeometry?.coordinates.orEmpty().size >= 2
    }
    if (routable.isEmpty()) {
        Box(
            modifier = modifier.background(MaterialTheme.colorScheme.surfaceContainerLow),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = stringResource(R.string.supplier_portal_dispatch_preview_map_text_route_preview_unavailable_until_optimizer_proposes_stops_with_co),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(16.dp),
            )
        }
        return
    }

    val context = LocalContext.current
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    val mapView = remember { MapView(context) }

    DisposableEffect(lifecycle, mapView) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> mapView.onStart()
                Lifecycle.Event.ON_RESUME -> mapView.onResume()
                Lifecycle.Event.ON_PAUSE -> mapView.onPause()
                Lifecycle.Event.ON_STOP -> mapView.onStop()
                Lifecycle.Event.ON_DESTROY -> mapView.onDestroy()
                else -> Unit
            }
        }
        lifecycle.addObserver(observer)
        onDispose {
            lifecycle.removeObserver(observer)
            mapView.onDestroy()
        }
    }

    LaunchedEffect(routable) {
        mapView.getMapAsync { map ->
            val callback: (Style) -> Unit = { style ->
                updateDispatchPreviewMap(style, map, routable)
            }
            if (map.style != null) {
                callback(map.style!!)
            } else {
                map.setStyle(Style.Builder().fromUri(STYLE_URL), callback)
            }
        }
    }

    AndroidView(
        factory = { mapView },
        modifier = modifier,
    )
}

private fun updateDispatchPreviewMap(
    style: Style,
    map: org.maplibre.android.maps.MapLibreMap,
    routes: List<DispatchProposedRoute>,
) {
    val boundsBuilder = LatLngBounds.Builder()
    var hasBounds = false

    routes.forEachIndexed { index, route ->
        val lineSourceId = "dispatch-preview-line-$index"
        val lineLayerId = "dispatch-preview-line-layer-$index"
        if (style.getLayer(lineLayerId) != null) {
            style.removeLayer(lineLayerId)
        }
        if (style.getSource(lineSourceId) != null) {
            style.removeSource(lineSourceId)
        }

        val coordinates = route.routeGeometry?.coordinates.orEmpty()
        if (coordinates.size >= 2) {
            val points = coordinates.map { Point.fromLngLat(it.lng, it.lat) }
            val feature = Feature.fromGeometry(LineString.fromLngLats(points))
            style.addSource(GeoJsonSource(lineSourceId, FeatureCollection.fromFeature(feature)))
            style.addLayer(
                LineLayer(lineLayerId, lineSourceId).withProperties(
                    PropertyFactory.lineColor(routeColor(index)),
                    PropertyFactory.lineWidth(4f),
                    PropertyFactory.lineOpacity(0.85f),
                    PropertyFactory.lineCap(Property.LINE_CAP_ROUND),
                    PropertyFactory.lineJoin(Property.LINE_JOIN_ROUND),
                ),
            )
            coordinates.forEach { coordinate ->
                boundsBuilder.include(LatLng(coordinate.lat, coordinate.lng))
                hasBounds = true
            }
        }
    }

    if (hasBounds) {
        val bounds = boundsBuilder.build()
        map.easeCamera(CameraUpdateFactory.newLatLngBounds(bounds, 96))
    }
}

private fun routeColor(index: Int): String = ROUTE_COLORS[index % ROUTE_COLORS.size]
