import re

with open("apps/backend-go/dispatch/binpack.go", "r") as f:
    content = f.read()

pattern = re.compile(r'for _, o := range remaining \{\n\t\t\t\t\t\tif chunkVU\+o\.VolumeVU <= maxFleetCap \|\| len\(chunk\) == 0 \{\n\t\t\t\t\t\t\tchunk = append\(chunk, o\)\n\t\t\t\t\t\t\tchunkVU \+= o\.VolumeVU\n\t\t\t\t\t\t\} else \{\n\t\t\t\t\t\t\tleftover = append\(leftover, o\)\n\t\t\t\t\t\t\}\n\t\t\t\t\t\}')

replacement = r"""for _, o := range remaining {
						if o.VolumeVU > maxFleetCap {
							chunks := int(o.VolumeVU / maxFleetCap)
							if o.VolumeVU > float64(chunks)*maxFleetCap {
								chunks++
							}
							split := SplitOrder{
								OriginalOrderID: o.OrderID,
								TotalVolumeVU:   o.VolumeVU,
								Reason:          "ORDER_EXCEEDS_FLEET_CAP",
								Chunks:          make([]OrderChunk, chunks),
							}
							volPerChunk := o.VolumeVU / float64(chunks)
							for i := 0; i < chunks; i++ {
								split.Chunks[i] = OrderChunk{
									ChunkIndex: i,
									VolumeVU:   volPerChunk,
								}
								sub := o
								sub.OrderID = fmt.Sprintf("%s-CHUNK-%d", o.OrderID, i)
								sub.VolumeVU = volPerChunk
								if chunkVU+volPerChunk <= maxFleetCap || len(chunk) == 0 {
									chunk = append(chunk, sub)
									chunkVU += volPerChunk
								} else {
									leftover = append(leftover, sub)
								}
							}
							result.Splits = append(result.Splits, split)
							continue
						}

						if chunkVU+o.VolumeVU <= maxFleetCap || len(chunk) == 0 {
							chunk = append(chunk, o)
							chunkVU += o.VolumeVU
						} else {
							leftover = append(leftover, o)
						}
					}"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/dispatch/binpack.go", "w") as f:
    f.write(content)
