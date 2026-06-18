import SwiftUI

private enum OrdersHubSurface: String, CaseIterable, Identifiable {
  case queue = "Orders"
  case dispatch = "Dispatch"

  var id: String { rawValue }
}

struct OrdersHubView: View {
  @Environment(\.horizontalSizeClass) private var horizontalSizeClass
  @State private var surface: OrdersHubSurface = .queue

  var body: some View {
    VStack(spacing: 0) {
      hubChrome
      Group {
        switch surface {
        case .queue:
          OrdersQueueView(embeddedInHub: true)
        case .dispatch:
            DispatchPreviewView(embeddedInHub: true)
        }
      }
      .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
    .background(SupplierTheme.background)
    .navigationTitle(surface.rawValue)
    .navigationBarTitleDisplayMode(.inline)
  }

  private var hubChrome: some View {
    VStack(spacing: SupplierTheme.spacingSM) {
      Picker("Surface", selection: $surface) {
        ForEach(OrdersHubSurface.allCases) { item in
          Text(item.rawValue).tag(item)
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
  @State private var vm = OrdersViewModel()

  var body: some View {
    Group {
      if horizontalSizeClass == .regular {
        splitContent
      } else if embeddedInHub {
        phoneContent
      } else {
        NavigationStack {
          phoneContent
            .navigationTitle("Orders")
            .toolbar { ordersToolbar }
        }
      }
    }
    .background(SupplierTheme.background)
    .task(id: vm.statusFilter) { await vm.load() }
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
            title: "Loading orders",
            message: "Fetching your supplier order queue."
          )
        } else if let error = vm.error, vm.orders.isEmpty {
          SupplierErrorView(message: error) { Task { await vm.load() } }
        } else if vm.orders.isEmpty {
          SupplierEmptyView(title: "No orders", message: "Nothing in this queue.")
        } else {
          List(vm.orders) { order in
            NavigationLink {
              OrderDetailPanel(order: order, vm: vm)
            } label: {
              OrderRow(order: order)
            }
          }
          .listStyle(.insetGrouped)
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
      .navigationTitle("Orders")
      .toolbar { ordersToolbar }
    } detail: {
      if let selection = vm.selection {
        OrderDetailPanel(order: selection, vm: vm)
      } else {
        ContentUnavailableView("Select an order", systemImage: "shippingbox")
      }
    }
  }

  private var filterTabs: some View {
    ScrollView(.horizontal, showsIndicators: false) {
      HStack(spacing: SupplierTheme.spacingSM) {
        ForEach(vm.filters, id: \.id) { filter in
          Button {
            vm.statusFilter = filter.id
          } label: {
            Text(filter.label)
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
          title: "Loading orders",
          message: "Fetching your supplier order queue."
        )
      } else if let error = vm.error, vm.orders.isEmpty {
        SupplierErrorView(message: error) { Task { await vm.load() } }
      } else if vm.orders.isEmpty {
        SupplierEmptyView(title: "No orders", message: "Nothing in this queue.")
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
      Button("Refresh", systemImage: "arrow.clockwise") {
        Task { await vm.load(silent: true) }
      }
      .labelStyle(.iconOnly)
    }
  }
}
