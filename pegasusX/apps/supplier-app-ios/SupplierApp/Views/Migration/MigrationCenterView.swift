import SwiftUI
import UniformTypeIdentifiers

struct MigrationCenterView: View {
    var showRedirectBanner: Bool = false

    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = MigrationViewModel()
    @State private var showFileImporter = false
    @State private var filePickError: String?
    @State private var activeMapperSession: MigrationSession?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingLG) {
                if showRedirectBanner {
                    redirectBanner
                }

                if let error = vm.error ?? filePickError {
                    errorBanner(error)
                }

                if let notice = vm.notice {
                    noticeBanner(notice)
                }

                projectHeaderCard

                if vm.loading && vm.projects.isEmpty {
                    SupplierLoadingView(title: "supplier_portal.migrations.title")
                } else if let tree = vm.tree {
                    uploadDropzoneCard(project: tree.project)
                    bundlesSection(tree.bundles)
                    sessionsSection(vm.sortedSessions, dag: tree.dag)
                    DryRunAndCutoverView(vm: vm)
                } else {
                    SupplierEmptyView(
                        title: "supplier_portal.migrations.title",
                        message: "supplier_portal.migrations.no_projects"
                    )
                }
            }
            .padding(SupplierTheme.spacingLG)
            .supplierReadableWidth()
        }
        .background(SupplierTheme.background)
        .navigationTitle(LocalizedStringKey("supplier_portal.migrations.title"))
        .refreshable { await vm.load() }
        .task { await vm.load() }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await vm.load(silent: silent) }
        }
        .sheet(isPresented: $vm.showCreateProjectSheet) {
            createProjectSheet
        }
        .sheet(item: $vm.activeReviewBundleTree) { bundleTree in
            AutoSortReviewSheet(vm: vm, bundleTree: bundleTree)
        }
        .sheet(item: $activeMapperSession, onDismiss: {
            vm.mappingSessionId = nil
        }) { _ in
            ColumnMapperSheet(vm: vm)
        }
        .fileImporter(
            isPresented: $showFileImporter,
            allowedContentTypes: [.item],
            allowsMultipleSelection: true
        ) { result in
            handleImportedFiles(result)
        }
    }

    private var redirectBanner: some View {
        HStack(alignment: .top, spacing: SupplierTheme.spacingSM) {
            Image(systemName: "arrow.triangle.2.circlepath.doc.on.clipboard")
                .foregroundStyle(SupplierTheme.primaryAction)
            Text(LocalizedStringKey("supplier_portal.migrations.catalog_redirect_notice"))
                .font(.subheadline)
                .foregroundStyle(SupplierTheme.label)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(SupplierTheme.spacingMD)
        .background(SupplierTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
    }

    private func errorBanner(_ message: String) -> some View {
        HStack(spacing: SupplierTheme.spacingSM) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(SupplierTheme.destructive)
            Text(message)
                .font(.caption)
                .foregroundStyle(SupplierTheme.label)
            Spacer()
            Button {
                vm.error = nil
                filePickError = nil
            } label: {
                Image(systemName: "xmark")
                    .font(.caption.bold())
            }
            .buttonStyle(.plain)
        }
        .padding(SupplierTheme.spacingMD)
        .background(SupplierTheme.destructive.opacity(0.12), in: RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
    }

    private func noticeBanner(_ message: String) -> some View {
        HStack(spacing: SupplierTheme.spacingSM) {
            Image(systemName: "checkmark.circle.fill")
                .foregroundStyle(SupplierTheme.success)
            Text(message)
                .font(.caption)
                .foregroundStyle(SupplierTheme.label)
            Spacer()
            Button {
                vm.notice = nil
            } label: {
                Image(systemName: "xmark")
                    .font(.caption.bold())
            }
            .buttonStyle(.plain)
        }
        .padding(SupplierTheme.spacingMD)
        .background(SupplierTheme.success.opacity(0.12), in: RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
    }

    private var projectHeaderCard: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
            HStack {
                SupplierSectionHeader(
                    title: "mobile_supplier.migrations.active_projects",
                    subtitle: "supplier_portal.migrations.subtitle"
                )
                Button {
                    vm.showCreateProjectSheet = true
                } label: {
                    Label("supplier_portal.migrations.create_button", systemImage: "plus")
                        .font(.subheadline.weight(.semibold))
                }
                .buttonStyle(.borderedProminent)
                .tint(SupplierTheme.primaryAction)
                .foregroundStyle(SupplierTheme.onPrimaryAction)
            }

            if !vm.projects.isEmpty {
                Picker("supplier_portal.migrations.project_name", selection: $vm.selectedProjectId) {
                    ForEach(vm.projects) { project in
                        Text("\(project.name) (\(project.status))").tag(Optional(project.projectId))
                    }
                }
                .pickerStyle(.menu)
                .onChange(of: vm.selectedProjectId) { _, newValue in
                    guard let newValue else { return }
                    Task { try? await vm.refreshTree(projectId: newValue) }
                }

                if let active = vm.tree?.project {
                    HStack(spacing: SupplierTheme.spacingSM) {
                        SupplierStatusBadge(text: active.status)
                        Text(active.mode)
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.secondaryLabel)
                        Spacer()
                        if let cutoverAt = active.cutoverAt, !cutoverAt.isEmpty {
                            SupplierStatusBadge(text: L10n.format("supplier_portal.migrations.status_live"), tint: SupplierTheme.success)
                        }
                    }
                }
            }
        }
        .supplierCard()
    }

    private func uploadDropzoneCard(project: MigrationProject) -> some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            SupplierSectionHeader(
                title: "supplier_portal.migrations.dropzone_title",
                subtitle: "supplier_portal.migrations.dropzone_subtitle"
            )

            Text(LocalizedStringKey("supplier_portal.migrations.resumable_hint"))
                .font(.caption)
                .foregroundStyle(SupplierTheme.secondaryLabel)

            if vm.uploading {
                ProgressView(
                    value: Double(vm.uploadedCount),
                    total: Double(max(vm.totalUploadCount, 1))
                )
                Text(L10n.format(
                    "supplier_portal.migrations.uploading_progress",
                    String(vm.uploadedCount),
                    String(vm.totalUploadCount)
                ))
                .font(.caption.monospacedDigit())
                .foregroundStyle(SupplierTheme.secondaryLabel)
            }

            Button {
                filePickError = nil
                showFileImporter = true
            } label: {
                Label("supplier_portal.migrations.select_files", systemImage: "doc.badge.plus")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .tint(SupplierTheme.primaryAction)
            .foregroundStyle(SupplierTheme.onPrimaryAction)
            .disabled(vm.uploading || vm.busy || project.status == "ARCHIVED")
        }
        .supplierCard()
    }

    @ViewBuilder
    private func bundlesSection(_ bundles: [MigrationBundleTree]) -> some View {
        if !bundles.isEmpty {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
                SupplierSectionHeader(title: "supplier_portal.migrations.sorter_title")

                ForEach(bundles) { bundleTree in
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                        HStack {
                            Text(L10n.format("supplier_portal.migrations.bundle_status", bundleTree.bundle.status))
                                .font(.subheadline.weight(.semibold))
                            Spacer()
                            SupplierStatusBadge(text: bundleTree.bundle.status)
                        }

                        let streamCount = bundleTree.bundle.sortSummary?.streams.count ?? 0
                        Text("\(bundleTree.artifacts.count) files • \(streamCount) streams")
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.secondaryLabel)

                        if bundleTree.bundle.sortSummary?.conflictingInventoryMode == true {
                            Text(LocalizedStringKey("supplier_portal.migrations.stock_vs_lots_warning"))
                                .font(.caption)
                                .foregroundStyle(SupplierTheme.warning)
                        }

                        Button {
                            vm.syncSortDrafts(from: bundleTree.bundle.sortSummary)
                            vm.activeReviewBundleTree = bundleTree
                        } label: {
                            Text(LocalizedStringKey("supplier_portal.migrations.sorter_title"))
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                    }
                    .padding(SupplierTheme.spacingMD)
                    .background(SupplierTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
                }
            }
            .supplierCard()
        }
    }

    @ViewBuilder
    private func sessionsSection(_ sessions: [MigrationSession], dag: MigrationDAGStatus) -> some View {
        if !sessions.isEmpty {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
                SupplierSectionHeader(title: "supplier_portal.migrations.mapper_title")

                ForEach(sessions) { session in
                    let isSelected = vm.selectedSession?.sessionId == session.sessionId
                    let packKey = session.pack ?? ""
                    let blockers = dag.blockedBy[packKey] ?? []

                    VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                        HStack(alignment: .top) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(LocalizedStringKey(MigrationPack.labelKey(for: packKey)))
                                    .font(.subheadline.weight(.semibold))
                                Text("\(session.fileName) • \(session.totalRows) rows")
                                    .font(.caption)
                                    .foregroundStyle(SupplierTheme.secondaryLabel)
                            }
                            Spacer()
                            SupplierStatusBadge(text: session.status)
                        }

                        if !blockers.isEmpty {
                            Text("Blocked by: \(blockers.joined(separator: ", "))")
                                .font(.caption2)
                                .foregroundStyle(SupplierTheme.warning)
                        }

                        HStack(spacing: SupplierTheme.spacingSM) {
                            Button {
                                vm.selectedSessionId = session.sessionId
                                PegasusHaptics.selection()
                            } label: {
                                Text(isSelected ? "Selected" : "Inspect")
                                    .font(.caption.weight(.semibold))
                            }
                            .buttonStyle(.bordered)

                            Button {
                                activeMapperSession = session
                                Task { await vm.openMapper(for: session.sessionId) }
                            } label: {
                                Label("supplier_portal.migrations.mapper_title", systemImage: "tablecells")
                                    .font(.caption.weight(.semibold))
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(SupplierTheme.primaryAction)
                            .foregroundStyle(SupplierTheme.onPrimaryAction)
                        }
                    }
                    .padding(SupplierTheme.spacingMD)
                    .background(
                        isSelected ? SupplierTheme.accentWarmSoft.opacity(0.4) : SupplierTheme.secondaryBackground,
                        in: RoundedRectangle(cornerRadius: SupplierTheme.radiusMD)
                    )
                    .overlay(
                        RoundedRectangle(cornerRadius: SupplierTheme.radiusMD)
                            .stroke(isSelected ? SupplierTheme.primaryAction : Color.clear, lineWidth: 1)
                    )
                }
            }
            .supplierCard()
        }
    }

    private var createProjectSheet: some View {
        NavigationStack {
            Form {
                Section("supplier_portal.migrations.project_name") {
                    TextField(
                        "supplier_portal.migrations.project_name_placeholder",
                        text: $vm.newProjectName
                    )
                }

                if !vm.warehouses.isEmpty {
                    Section("supplier_portal.migrations.warehouse_label") {
                        Picker("supplier_portal.migrations.warehouse_label", selection: $vm.newProjectWarehouseId) {
                            Text("—").tag("")
                            ForEach(vm.warehouses) { wh in
                                let label = wh.address.isEmpty ? wh.name : "\(wh.name) (\(wh.address))"
                                Text(label).tag(wh.id)
                            }
                        }
                    }
                }
            }
            .navigationTitle(LocalizedStringKey("supplier_portal.migrations.create_project"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { vm.showCreateProjectSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        Task { await vm.createProject() }
                    } label: {
                        Text(LocalizedStringKey(
                            vm.busy ? "supplier_portal.migrations.creating" : "supplier_portal.migrations.create_button"
                        ))
                    }
                    .disabled(vm.newProjectName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || vm.busy)
                }
            }
        }
    }

    private func handleImportedFiles(_ result: Result<[URL], Error>) {
        switch result {
        case .failure(let err):
            filePickError = err.localizedDescription
            PegasusHaptics.error()
        case .success(let urls):
            guard !urls.isEmpty else { return }
            Task {
                do {
                    let artifacts = try await Task.detached(priority: .userInitiated) {
                        var collected: [PickedMigrationArtifact] = []
                        for url in urls.prefix(50) {
                            let scoped = url.startAccessingSecurityScopedResource()
                            defer {
                                if scoped { url.stopAccessingSecurityScopedResource() }
                            }
                            let data = try Data(contentsOf: url)
                            let digest = MigrationService.sha256Hex(of: data)
                            collected.append(
                                PickedMigrationArtifact(
                                    fileName: url.lastPathComponent,
                                    data: data,
                                    sha256: digest
                                )
                            )
                        }
                        return collected
                    }.value
                    await vm.uploadPickedFiles(artifacts)
                } catch {
                    filePickError = error.localizedDescription
                    PegasusHaptics.error()
                }
            }
        }
    }
}
