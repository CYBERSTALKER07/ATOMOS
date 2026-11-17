import SwiftUI

private enum OrdersHubSurface: String, CaseIterable, Identifiable {
  case queue = "mobile_supplier.chrome.orders"
  case dispatch = "mobile_supplier.chrome.dispatch"

  var id: String { rawValue }
}

struct OrdersHubView: View {
  @Environment(\.horizontalSizeClass) private var horizontalSizeClass
  var initialCommandStatus: String? = nil
  @State private var surface: OrdersHubSurface = .queue

  var body: some View {
    VStack(spacing: 0) {
      hubChrome
      Group {
        switch surface {
        case .queue:
          OrdersQueueView(embeddedInHub: true, initialCommandStatus: initialCommandStatus)
        case .dispatch:
            DispatchPreviewView(embeddedInHub: true)
        }
      }
      .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
    .background(SupplierTheme.background)
    .navigationTitle(LocalizedStringKey(surface.rawValue))
    .navigationBarTitleDisplayMode(.inline)
  }

  private var hubChrome: some View {
    VStack(spacing: SupplierTheme.spacingSM) {
      Picker("mobile_supplier.chrome.surface", selection: $surface) {
        ForEach(OrdersHubSurface.allCases) { item in
          Text(LocalizedStringKey(item.rawValue)).tag(item)
        }
      }
      .pickerStyle(.segmented)
      .padding(.horizontal)
      .padding(.top, SupplierTheme.spacingSM)
    }
    .background(SupplierTheme.background)
  }
}

struct OrdersQueueView: View {
  @Environment(\.horizontalSizeClass) private var horizontalSizeClass
  @Environment(SupplierRealtimeHub.self) private var realtimeHub
  var embeddedInHub: Bool = false
  var initialCommandStatus: String? = nil
  @State private var vm = OrdersViewModel()

  var body: some View {
    Group {
      if horizontalSizeClass == .regular {
        splitContent
      } else if embeddedInHub {
        phoneContent
      } else {
        phoneContent
          .navigationTitle("portal.nav.orders")
          .toolbar { ordersToolbar }
      }
    }
    .background(SupplierTheme.background)
    .onAppear {
      if let initialCommandStatus {
        vm.applyCommandStatus(initialCommandStatus)
      }
    }
    .task(id: vm.loadIdentity) { await vm.load() }
    .onChange(of: realtimeHub.refreshEpoch) { _, _ in
      Task { await vm.load(silent: true) }
    }
    .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
      Task { await vm.load(silent: true) }
    }
  }

  private var phoneContent: some View {
    VStack(spacing: 0) {
      filterTabs
      Group {
        if vm.loading && vm.orders.isEmpty {
          SupplierLoadingView(
            title: "mobile_supplier.loading.orders",
            message: "mobile_supplier.loading.orders_message"
          )
        } else if let error = vm.error, vm.orders.isEmpty {
          SupplierErrorView(message: error) { Task { await vm.load() } }
        } else if vm.orders.isEmpty {
          SupplierEmptyView(title: "mobile_supplier.empty.no_orders", message: "mobile_supplier.empty.nothing_in_queue")
        } else {
          ResponsiveGridContentWrapper {
            ForEach(vm.orders) { order in
              NavigationLink {
                OrderDetailPanel(order: order, vm: vm)
              } label: {
                OrderRow(
                  order: order,
                  showWarehouseMenu: vm.canWarehouseOps(for: order),
                  onDelay: nil,
                  onReject: nil,
                )
              }
            }
          }
        }
      }
      .refreshable { await vm.load(silent: true) }
    }
  }

  private var splitContent: some View {
    NavigationSplitView {
      VStack(spacing: 0) {
        filterTabs
        ordersList
      }
      .navigationTitle("portal.nav.orders")
      .toolbar { ordersToolbar }
    } detail: {
      if let selection = vm.selection {
        OrderDetailPanel(order: selection, vm: vm)
      } else {
        ContentUnavailableView("mobile_supplier.chrome.select_an_order", systemImage: "shippingbox")
      }
    }
  }

  private var filterTabs: some View {
    ScrollView(.horizontal, showsIndicators: false) {
      HStack(spacing: SupplierTheme.spacingSM) {
        if let commandStatus = vm.commandStatus {
          HStack {
            Text(L10n.format("mobile_supplier.chrome.filtered_by", commandStatus.replacingOccurrences(of: "_", with: " ")))
              .font(.caption)
              .accessibilityIdentifier("gs-u-command-filter")
            Button("mobile_supplier.chrome.clear") { vm.setCoarseFilter(vm.statusFilter) }
              .font(.caption.bold())
          }
          .padding(.horizontal)
        }
        ForEach(vm.filters, id: \.id) { filter in
          Button {
            vm.setCoarseFilter(filter.id)
          } label: {
            Text(LocalizedStringKey(filter.label))
              .font(.subheadline.weight(vm.statusFilter == filter.id ? .semibold : .regular))
              .padding(.horizontal, 12)
              .padding(.vertical, 8)
              .background(vm.statusFilter == filter.id ? Color.primary : Color.clear)
              .foregroundStyle(vm.statusFilter == filter.id ? Color(.systemBackground) : Color.primary)
              .clipShape(Capsule())
              .overlay {
                Capsule().strokeBorder(Color.primary.opacity(0.25), lineWidth: vm.statusFilter == filter.id ? 0 : 1)
              }
          }
          .buttonStyle(.plain)
        }
      }
      .padding(.horizontal)
      .padding(.vertical, SupplierTheme.spacingSM)
    }
    .background(SupplierTheme.background)
  }

  private var ordersList: some View {
    Group {
      if vm.loading && vm.orders.isEmpty {
        SupplierLoadingView(
          title: "mobile_supplier.loading.orders",
          message: "mobile_supplier.loading.orders_message"
        )
      } else if let error = vm.error, vm.orders.isEmpty {
        SupplierErrorView(message: error) { Task { await vm.load() } }
      } else if vm.orders.isEmpty {
        SupplierEmptyView(title: "mobile_supplier.empty.no_orders", message: "mobile_supplier.empty.nothing_in_queue")
      } else {
        List(vm.orders, selection: $vm.selection) { order in
          OrderRow(order: order)
            .tag(order)
        }
        .listStyle(.sidebar)
      }
    }
    .refreshable { await vm.load(silent: true) }
  }

  @ToolbarContentBuilder
  private var ordersToolbar: some ToolbarContent {
    ToolbarItem(placement: .topBarTrailing) {
      Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
        Task { await vm.load(silent: true) }
      }
      .labelStyle(.iconOnly)
    }
  }
}
