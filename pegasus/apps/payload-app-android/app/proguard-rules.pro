# ProGuard/R8 rules for PegasusPayload release builds.
-keepattributes Signature, *Annotation*, EnclosingMethod, InnerClasses
-keep class kotlinx.serialization.** { *; }
-keep class com.pegasus.payload.data.model.** { *; }
-dontwarn kotlinx.serialization.**
-dontwarn org.bouncycastle.**
-dontwarn org.conscrypt.**
-dontwarn org.openjsse.**
