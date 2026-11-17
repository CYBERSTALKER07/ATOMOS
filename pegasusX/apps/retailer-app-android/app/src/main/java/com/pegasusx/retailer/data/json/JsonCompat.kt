package com.pegasusx.retailer.data.json

import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.double
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.long

/**
 * Gson-style accessors over kotlinx.serialization.json.
 *
 * Several screens were written against Gson's JsonElement API before the app moved to
 * kotlinx.serialization. These shims keep those call sites working with identical
 * semantics (throwing on wrong type, same as Gson). New code should use the kotlinx
 * accessors (jsonObject/jsonPrimitive/booleanOrNull/...) directly.
 */
val JsonElement.asJsonObject: JsonObject get() = jsonObject
val JsonElement.asJsonArray: JsonArray get() = jsonArray
val JsonElement.asString: String get() = jsonPrimitive.content
val JsonElement.asBoolean: Boolean get() = jsonPrimitive.boolean
val JsonElement.asInt: Int get() = jsonPrimitive.int
val JsonElement.asLong: Long get() = jsonPrimitive.long
val JsonElement.asDouble: Double get() = jsonPrimitive.double
val JsonElement.isJsonNull: Boolean get() = this is JsonNull

fun JsonObject.getAsJsonArray(member: String): JsonArray? = this[member]?.let { it as? JsonArray }

val JsonElement?.asStringOrNull: String? get() = (this as? JsonPrimitive)?.contentOrNull
val JsonElement?.asDoubleOrNull: Double? get() = (this as? JsonPrimitive)?.doubleOrNull
