import Charts
import SwiftUI

public struct InfraHealthView: View {
    @ObservedObject var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                // Section Title
                VStack(alignment: .leading, spacing: 4) {
                    Text("Cloud & Multi-Cell Infrastructure Telemetry")
                        .font(.title2)
                        .fontWeight(.bold)
                    Text("Direct instrumentation into distributed Cloud Spanner quorum, Kafka broker lag, and Redis priority semaphores.")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }

                // Cloud Spanner Latency Chart (Swift Charts)
                if let spanner = viewModel.spanner {
                    VStack(alignment: .leading, spacing: 14) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Cloud Spanner Multi-Region P99 Latency (ms)")
                                    .font(.headline)
                                Text("True-Time synchronized distributed transaction latency and replica offload.")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                            Spacer()
                            HStack(spacing: 6) {
                                Circle()
                                    .fill(Color.teal)
                                    .frame(width: 8, height: 8)
                                Text("Replica Stale Reads: \(String(format: "%.1f", spanner.staleReadOffloadPercent))% Offloaded")
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundColor(.secondary)
                            }
                        }

                        HStack(spacing: 16) {
                            latencyStatPill(label: "Read P99", value: "\(String(format: "%.1f", spanner.p99ReadMs)) ms", color: .blue)
                            latencyStatPill(label: "Write P99", value: "\(String(format: "%.1f", spanner.p99WriteMs)) ms", color: .orange)
                            latencyStatPill(label: "Commit P99", value: "\(String(format: "%.1f", spanner.p99CommitMs)) ms", color: .purple)
                            Spacer()
                        }

                        Chart {
                            ForEach(spanner.hourlyHistory) { point in
                                AreaMark(
                                    x: .value("Hour", point.hour),
                                    y: .value("Read P99", point.readMs)
                                )
                                .foregroundStyle(LinearGradient(colors: [Color.blue.opacity(0.18), Color.blue.opacity(0.01)], startPoint: .top, endPoint: .bottom))
                                .interpolationMethod(.catmullRom)

                                LineMark(
                                    x: .value("Hour", point.hour),
                                    y: .value("Read P99", point.readMs)
                                )
                                .foregroundStyle(Color.blue)
                                .interpolationMethod(.catmullRom)
                                .lineStyle(StrokeStyle(lineWidth: 2.2))

                                LineMark(
                                    x: .value("Hour", point.hour),
                                    y: .value("Write P99", point.writeMs)
                                )
                                .foregroundStyle(Color.orange)
                                .interpolationMethod(.catmullRom)
                                .lineStyle(StrokeStyle(lineWidth: 2.0))

                                LineMark(
                                    x: .value("Hour", point.hour),
                                    y: .value("Commit P99", point.commitMs)
                                )
                                .foregroundStyle(Color.purple)
                                .interpolationMethod(.catmullRom)
                                .lineStyle(StrokeStyle(lineWidth: 1.8, dash: [4, 4]))
                            }
                        }
                        .frame(height: 190)
                        .accessibilityLabel("Cloud Spanner Multi-Region P99 Latency Chart")
                        .accessibilityValue("Read \(String(format: "%.1f", spanner.p99ReadMs)) ms, Write \(String(format: "%.1f", spanner.p99WriteMs)) ms, Commit \(String(format: "%.1f", spanner.p99CommitMs)) ms")
                    }
                    .padding(18)
                    .background(.ultraThinMaterial)
                    .overlay(
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .stroke(
                                LinearGradient(
                                    colors: [Color.primary.opacity(0.12), Color.primary.opacity(0.02)],
                                    startPoint: .topLeading,
                                    endPoint: .bottomTrailing
                                ),
                                lineWidth: 1
                            )
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                    .shadow(color: Color.black.opacity(0.05), radius: 8, x: 0, y: 3)
                }

                // Kafka Partition Lag Matrix
                VStack(alignment: .leading, spacing: 12) {
                    HStack {
                        Text("Distributed Kafka Partition Lag & DLQ Backlog")
                            .font(.headline)
                        Spacer()
                        Text("3 Outbox Topics")
                            .font(.system(.caption, design: .monospaced))
                            .foregroundColor(.secondary)
                    }

                    VStack(spacing: 8) {
                        ForEach(viewModel.kafkaPartitions) { kafka in
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack(spacing: 8) {
                                        Text(kafka.topic)
                                            .font(.system(.subheadline, design: .monospaced))
                                            .fontWeight(.bold)
                                        Button {
                                            viewModel.copyToClipboard(kafka.topic, label: "Topic")
                                        } label: {
                                            Image(systemName: "doc.on.doc")
                                                .font(.caption2)
                                                .foregroundColor(.secondary)
                                        }
                                        .buttonStyle(.plain)
                                    }
                                    Text("Consumer Group: \(kafka.consumerGroup) • \(kafka.partitions) Partitions")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                HStack(spacing: 20) {
                                    VStack(alignment: .trailing, spacing: 2) {
                                        Text("\(kafka.maxLagMs) ms")
                                            .font(.system(.subheadline, design: .monospaced))
                                            .fontWeight(.bold)
                                            .monospacedDigit()
                                        Text("Max Lag")
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                    VStack(alignment: .trailing, spacing: 2) {
                                        Text("\(kafka.dlqCount) msgs")
                                            .font(.system(.subheadline, design: .monospaced))
                                            .fontWeight(.bold)
                                            .foregroundColor(kafka.dlqCount > 0 ? .red : .green)
                                            .monospacedDigit()
                                        Text("Poison DLQ")
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                }
                            }
                            .padding(14)
                            .background(.ultraThinMaterial)
                            .overlay(
                                RoundedRectangle(cornerRadius: 12, style: .continuous)
                                    .stroke(Color.primary.opacity(0.06), lineWidth: 1)
                            )
                            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Kafka Topic: \(kafka.topic). Max Lag: \(kafka.maxLagMs) milliseconds. Poison DLQ: \(kafka.dlqCount) messages.")
                        }
                    }
                }

                // Redis Priority Permit Semaphores
                VStack(alignment: .leading, spacing: 10) {
                    Text("Redis Priority Semaphore Permits")
                        .font(.headline)

                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                        ForEach(viewModel.redisPools) { pool in
                            VStack(alignment: .leading, spacing: 8) {
                                HStack {
                                    Text(pool.poolName)
                                        .font(.system(.subheadline, design: .monospaced))
                                        .fontWeight(.semibold)
                                    Spacer()
                                    Text("\(pool.activePermits) / \(pool.maxPermits)")
                                        .font(.system(.caption, design: .monospaced))
                                        .foregroundColor(.secondary)
                                }
                                ProgressView(value: Double(pool.activePermits), total: Double(pool.maxPermits))
                                    .tint(pool.utilizationPercent > 80 ? .orange : .blue)
                                HStack {
                                    Text("Queue wait: \(String(format: "%.1f", pool.queueWaitMs))ms")
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                    Spacer()
                                    Text("\(String(format: "%.0f", pool.utilizationPercent))% utilized")
                                        .font(.caption2)
                                        .fontWeight(.semibold)
                                        .foregroundColor(pool.utilizationPercent > 80 ? .orange : .green)
                                }
                            }
                            .padding(12)
                            .background(Color(NSColor.controlBackgroundColor))
                            .cornerRadius(8)
                        }
                    }
                }

                Spacer()
            }
            .padding(24)
        }
    }

    private func latencyStatPill(label: String, value: String, color: Color) -> some View {
        HStack(spacing: 6) {
            Circle()
                .fill(color)
                .frame(width: 8, height: 8)
            VStack(alignment: .leading, spacing: 1) {
                Text(label)
                    .font(.caption2)
                    .foregroundColor(.secondary)
                Text(value)
                    .font(.system(.subheadline, design: .monospaced))
                    .fontWeight(.bold)
                    .monospacedDigit()
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}
