import SwiftUI

enum LabIcon: String, CaseIterable {
    case home
    case cart
    case truck
    case check
    case user
    case bell
    case search
    case history
    case inbox
    case procurement
    case ai
    case qr
    case close
    case menu
    case plus
    case minus
    case chevronRight
    case settings
    case logout

    var systemName: String {
        switch self {
        case .home: "house"
        case .cart: "cart"
        case .truck: "shippingbox"
        case .check: "checkmark.circle"
        case .user: "person"
        case .bell: "bell"
        case .search: "magnifyingglass"
        case .history: "clock"
        case .inbox: "tray"
        case .procurement: "chart.bar"
        case .ai: "sparkles"
        case .qr: "qrcode"
        case .close: "xmark"
        case .menu: "line.3.horizontal"
        case .plus: "plus"
        case .minus: "minus"
        case .chevronRight: "chevron.right"
        case .settings: "gearshape"
        case .logout: "rectangle.portrait.and.arrow.right"
        }
    }

    var filledSystemName: String {
        switch self {
        case .home: "house.fill"
        case .cart: "cart.fill"
        case .truck: "shippingbox.fill"
        case .check: "checkmark.circle.fill"
        case .user: "person.fill"
        case .bell: "bell.fill"
        case .search: "magnifyingglass"
        case .history: "clock.fill"
        case .inbox: "tray.fill"
        case .procurement: "chart.bar.fill"
        case .ai: "sparkles"
        case .qr: "qrcode"
        case .close: "xmark.circle.fill"
        case .menu: "line.3.horizontal"
        case .plus: "plus.circle.fill"
        case .minus: "minus.circle.fill"
        case .chevronRight: "chevron.right"
        case .settings: "gearshape.fill"
        case .logout: "rectangle.portrait.and.arrow.right"
        }
    }

    func image(filled: Bool = false) -> Image {
        Image(systemName: filled ? filledSystemName : systemName)
    }
}
