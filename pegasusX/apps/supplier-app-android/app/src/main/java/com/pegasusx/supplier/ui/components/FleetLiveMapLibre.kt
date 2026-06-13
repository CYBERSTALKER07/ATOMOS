package com.pegasusx.supplier.ui.components

import android.graphics.Color
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.pegasusx.supplier.data.model.SupplierFleetLiveRoute
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style
import org.maplibre.android.style.layers.CircleLayer
import org.maplibre.android.style.layers.LineLayer
import org.maplibre.android.style.layers.Property
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.LineString
import org.maplibre.geojson.Point

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
fun FleetLiveMapLibre(
    routes: List<SupplierFleetLiveRoute>,
    modifier: Modifier = Modifier,
    animateDrivers: Boolean = true,
) {
    val context = LocalContext.current
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    val mapView = remember { MapView(context) }
    val animator = remember { FleetDriverMarkerAnimator() }
    var animatedDrivers by remember { mutableStateOf<List<AnimatedDriverPoint>>(emptyList()) }

    LaunchedEffect(routes, animateDrivers) {
        if (!animateDrivers) {
            animatedDrivers = emptyList()
            return@LaunchedEffect
        }
        animator.updateTargets(routes, ::routeColor)
        while (isActive) {
            animatedDrivers = animator.snapshot(System.currentTimeMillis())
            delay(16L)
        }
    }

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

    LaunchedEffect(routes, animatedDrivers, animateDrivers) {
        mapView.getMapAsync { map ->
            val callback: (Style) -> Unit = { style ->
                updateFleetMap(style, map, routes, if (animateDrivers) animatedDrivers else null)
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

private fun updateFleetMap(
    style: Style,
    map: org.maplibre.android.maps.MapLibreMap,
    routes: List<SupplierFleetLiveRoute>,
    animatedDrivers: List<AnimatedDriverPoint>?,
) {
    val boundsBuilder = LatLngBounds.Builder()
    var hasBounds = false

    routes.forEachIndexed { index, route ->
        val lineSourceId = "fleet-line-${route.manifestId}"
        val lineLayerId = "fleet-line-layer-${route.manifestId}"
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

    val driverFeatures = if (animatedDrivers != null) {
        animatedDrivers.map { point ->
            val feature = Feature.fromGeometry(Point.fromLngLat(point.lng, point.lat))
            feature.addStringProperty("color", point.color)
            feature.addNumberProperty("stale", if (point.stale) 1 else 0)
            boundsBuilder.include(LatLng(point.lat, point.lng))
            hasBounds = true
            feature
        }
    } else {
        routes.mapIndexedNotNull { index, route ->
            val location = route.driverLocation
            if (!route.liveLocationAvailable || location == null) {
                return@mapIndexedNotNull null
            }
            val lat = if (location.latitude != 0.0) location.latitude else location.lat
            val lng = if (location.longitude != 0.0) location.longitude else location.lng
            val feature = Feature.fromGeometry(Point.fromLngLat(lng, lat))
            feature.addStringProperty("color", routeColor(index))
            feature.addNumberProperty("stale", if (route.locationStale == true) 1 else 0)
            boundsBuilder.include(LatLng(lat, lng))
            hasBounds = true
            feature
        }
    }

    if (style.getLayer("fleet-driver-points") != null) {
        style.removeLayer("fleet-driver-points")
    }
    if (style.getSource("fleet-driver-points") != null) {
        style.removeSource("fleet-driver-points")
    }
    if (driverFeatures.isNotEmpty()) {
        style.addSource(GeoJsonSource("fleet-driver-points", FeatureCollection.fromFeatures(driverFeatures)))
        style.addLayer(
            CircleLayer("fleet-driver-points", "fleet-driver-points").withProperties(
                PropertyFactory.circleRadius(7f),
                PropertyFactory.circleColor(ExpressionColorFromProperty()),
                PropertyFactory.circleStrokeColor(Color.WHITE),
                PropertyFactory.circleStrokeWidth(2f),
                PropertyFactory.circleOpacity(
                    org.maplibre.android.style.expressions.Expression.switchCase(
                        org.maplibre.android.style.expressions.Expression.eq(
                            org.maplibre.android.style.expressions.Expression.get("stale"),
                            org.maplibre.android.style.expressions.Expression.literal(1),
                        ),
                        org.maplibre.android.style.expressions.Expression.literal(0.45f),
                        org.maplibre.android.style.expressions.Expression.literal(1f),
                    ),
                ),
            ),
        )
    }

    if (hasBounds) {
        val bounds = boundsBuilder.build()
        map.easeCamera(CameraUpdateFactory.newLatLngBounds(bounds, 96))
    }
}

private fun routeColor(index: Int): String = ROUTE_COLORS[index % ROUTE_COLORS.size]

private fun ExpressionColorFromProperty() =
    org.maplibre.android.style.expressions.Expression.toColor(
        org.maplibre.android.style.expressions.Expression.get("color"),
    )
