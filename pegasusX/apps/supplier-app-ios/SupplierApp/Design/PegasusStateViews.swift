import Observation
import SwiftUI

public struct PegasusLoadingView: View {
  public let title: String
  public var message: String = "mobile_supplier.loading.latest"

  @Environment(\.accessibilityReduceMotion) private var reduceMotion
  @State private var animating = false

  public init(title: String, message: String = "mobile_supplier.loading.latest") {
    self.title = title
    self.message = message
  }

  public var body: some View {
    VStack(spacing: PegasusMonochromeTheme.spacingLG) {
      ZStack {
        Circle()
          .fill(PegasusMonochromeTheme.tertiaryBackground)
          .frame(width: 72, height: 72)
          .scaleEffect(reduceMotion ? 1.0 : (animating ? 1.04 : 0.96))
        ProgressView()
          .controlSize(.regular)
      }

      VStack(spacing: PegasusMonochromeTheme.spacingSM) {
        Text(LocalizedStringKey(title))
          .font(.title3.bold())
        Text(LocalizedStringKey(message))
          .font(.body)
          .foregroundStyle(.secondary)
          .multilineTextAlignment(.center)
      }
    }
    .frame(maxWidth: .infinity, minHeight: 200)
    .padding(PegasusMonochromeTheme.spacingXL)
    .onAppear {
      guard !reduceMotion else { return }
      withAnimation(PegasusAnim.smooth.repeatForever(autoreverses: true)) {
        animating = true
      }
    }
  }
}

public struct PegasusErrorView: View {
  public let message: String
  public let retry: () -> Void

  public init(message: String, retry: @escaping () -> Void) {
    self.message = message
    self.retry = retry
  }

  public var body: some View {
    ContentUnavailableView {
      Label("supplier_portal.residual.text.unable_to_load", systemImage: "exclamationmark.triangle")
    } description: {
      Text(message)
    } actions: {
      Button("common.action.retry", action: retry)
        .buttonStyle(.borderedProminent)
    }
    .frame(maxWidth: .infinity, minHeight: 200)
  }
}

@Observable
@MainActor
final class SupplierStatusCenter {
  static let shared = SupplierStatusCenter()

  var notice: String?

  func clear() {
    notice = nil
  }
}

@MainActor
enum SupplierScreenLoad {
  static func begin(silent: Bool, loading: inout Bool, error: inout String?) {
    if !silent {
      loading = true
      error = nil
      SupplierStatusCenter.shared.notice = nil
    }
  }

  static func succeed(error: inout String?) {
    error = nil
    SupplierStatusCenter.shared.notice = nil
  }

  static func fail(
    silent: Bool,
    hasContent: Bool,
    underlying: Error,
    error: inout String?,
    noun: String
  ) {
    fail(
      silent: silent,
      hasContent: hasContent,
      underlyingMessage: underlying.localizedDescription,
      error: &error,
      noun: noun
    )
  }

  static func fail(
    silent: Bool,
    hasContent: Bool,
    underlyingMessage: String,
    error: inout String?,
    noun: String
  ) {
    let outcome = OrdersLoadFailure.message(
      offline: !NetworkLifecycleEngine.shared.isOnline,
      silent: silent,
      hasOrders: hasContent,
      underlying: underlyingMessage,
      noun: noun
    )
    if !silent || outcome.error != nil {
      error = outcome.error
    }
    if let notice = outcome.notice {
      SupplierStatusCenter.shared.notice = notice
    } else if !silent || outcome.error != nil {
      SupplierStatusCenter.shared.notice = nil
    }
  }
}

@MainActor
enum SupplierActionFailure {
  static func record(underlying: Error, hasContent: Bool, error: inout String?) {
    let message = NetworkLifecycleEngine.shared.isOnline
      ? underlying.localizedDescription
      : L10n.format("mobile_supplier.chrome.offline_try_again")
    if hasContent {
      SupplierStatusCenter.shared.notice = message
    } else {
      error = message
    }
  }
}
