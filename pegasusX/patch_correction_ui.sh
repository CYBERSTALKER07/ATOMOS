sed -i '' '/DriverSectionTitle(title = "MANIFEST ITEMS/i\
                    if (state.isPartial) {\
                        item {\
                            Card(\
                                modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),\
                                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)\
                            ) {\
                                Column(modifier = Modifier.padding(16.dp)) {\
                                    Text(\
                                        text = "Partial Order Split",\
                                        style = MaterialTheme.typography.titleMedium,\
                                        fontWeight = FontWeight.Bold,\
                                        color = MaterialTheme.colorScheme.onPrimaryContainer\
                                    )\
                                    Spacer(modifier = Modifier.height(4.dp))\
                                    Text(\
                                        text = "This order is split across multiple trucks. Press Start Transit when you are heading to this route to notify the other driver.",\
                                        style = MaterialTheme.typography.bodyMedium,\
                                        color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.8f)\
                                    )\
                                    Spacer(modifier = Modifier.height(12.dp))\
                                    Button(\
                                        onClick = { showConfirmDialog = true; /* We will use showConfirmDialog but we need a different one for Start Transit */ },\
                                        modifier = Modifier.fillMaxWidth()\
                                    ) {\
                                        Text("Start Transit")\
                                    }\
                                }\
                            }\
                        }\
                    }\
' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt
