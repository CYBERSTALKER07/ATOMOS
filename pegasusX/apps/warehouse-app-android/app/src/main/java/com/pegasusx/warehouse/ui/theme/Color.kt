package com.pegasusx.warehouse.ui.theme

import androidx.compose.ui.graphics.Color

// ── M3 Neutral Tonal Palette (Warm Monochrome - Sampled from Reference Image) ──
val Neutral0 = Color(0xFF000000)
val Neutral4 = Color(0xFF0A0A0C)
val Neutral6 = Color(0xFF0E0E10) // Dark mode background canvas
val Neutral10 = Color(0xFF111113) // Primary text & solid black elements / FAB / chart bars
val Neutral12 = Color(0xFF18181B) // Dark mode elevated card surface
val Neutral17 = Color(0xFF27272A) // Dark mode container / icon tile
val Neutral20 = Color(0xFF2E2E32)
val Neutral22 = Color(0xFF35353A)
val Neutral24 = Color(0xFF3F3F46)
val Neutral30 = Color(0xFF52525B)
val Neutral40 = Color(0xFF71717A) // Secondary label & section headers ("Latest", "Monday")
val Neutral50 = Color(0xFF71717A)
val Neutral60 = Color(0xFFA1A1AA) // Tertiary label & timestamps & axis ("10 Mar 2026")
val Neutral70 = Color(0xFFD4D4D8)
val Neutral80 = Color(0xFFE4E4E7) // Subtle divider lines
val Neutral87 = Color(0xFFE4E4E7)
val Neutral90 = Color(0xFFE8E8EC)
val Neutral92 = Color(0xFFE2E2E6) // Icon tile / container squircle (Light)
val Neutral94 = Color(0xFFEAEAEB) // Pill buttons / secondary controls
val Neutral95 = Color(0xFFEDEDF0) // Warm canvas background (Light)
val Neutral96 = Color(0xFFF4F4F6)
val Neutral98 = Color(0xFFFBFBFB) // Crisp elevated card
val Neutral100 = Color(0xFFFFFFFF)

// ── Semantic Status Colors ──
val Destructive = Color(0xFFEF4444)
val Success = Color(0xFF22C55E)
val Warning = Color(0xFFF59E0B)
val Live = Color(0xFF3B82F6)

// ── M3 Warehouse Color Roles (Monochrome Aligned) ──
val WarehousePrimary = Neutral10
val WarehouseOnPrimary = Neutral100
val WarehousePrimaryContainer = Neutral94
val WarehouseOnPrimaryContainer = Neutral10

val WarehouseSecondary = Neutral40
val WarehouseOnSecondary = Neutral100
val WarehouseSecondaryContainer = Neutral92
val WarehouseOnSecondaryContainer = Neutral10

val WarehouseTertiary = Neutral30
val WarehouseOnTertiary = Neutral100
val WarehouseTertiaryContainer = Neutral90
val WarehouseOnTertiaryContainer = Neutral10

val WarehouseError = Destructive
val WarehouseOnError = Neutral100
val WarehouseErrorContainer = Destructive.copy(alpha = 0.12f)
val WarehouseOnErrorContainer = Destructive

val WarehouseSurfaceLight = Neutral100
val WarehouseOnSurfaceLight = Neutral10
val WarehouseSurfaceVariantLight = Neutral94
val WarehouseOnSurfaceVariantLight = Neutral40
val WarehouseOutlineLight = Neutral70
val WarehouseSurfaceContainerLight = Neutral96
val WarehouseSurfaceContainerHighLight = Neutral94

val WarehouseSurfaceDark = Neutral10
val WarehouseOnSurfaceDark = Neutral90
val WarehouseSurfaceVariantDark = Neutral17
val WarehouseOnSurfaceVariantDark = Neutral60
val WarehouseOutlineDark = Neutral40
val WarehouseSurfaceContainerDark = Neutral12
val WarehouseSurfaceContainerHighDark = Neutral17
