import {
  LayoutDashboard,
  ShoppingCart,
  PackageSearch,
  MapPin,
  Container,
  Activity,
  BarChart3,
  Settings,
  Bell,
  LogOut,
  Menu,
  ChevronRight,
  Sun,
  Moon,
  Monitor,
  ArrowLeft,
  AlertTriangle,
  RefreshCw,
  ChevronLeft,
  Plus,
  CircleCheck,
  CircleAlert,
  Store,
  CreditCard,
  Users,
  type LucideIcon,
} from "lucide-react";

const iconMap: Record<string, LucideIcon> = {
  dashboard: LayoutDashboard,
  orders: ShoppingCart,
  catalog: PackageSearch,
  tracking: MapPin,
  dock: Container,
  procurement: Activity,
  insights: BarChart3,
  notifications: Bell,
  settings: Settings,
  store: Store,
  cards: CreditCard,
  family: Users,
  logout: LogOut,
  menu: Menu,
  chevronR: ChevronRight,
  arrowBack: ArrowLeft,
  warning: AlertTriangle,
  lightMode: Sun,
  darkMode: Moon,
  autoMode: Monitor,
  refresh: RefreshCw,
  left: ChevronLeft,
  right: ChevronRight,
  add: Plus,
  check_circle: CircleCheck,
  error: CircleAlert,
  verified: CircleCheck,
  arrow_forward: ChevronRight,
};

export default function Icon({
  name,
  size = 24,
  className = "",
}: {
  name: string;
  size?: number;
  className?: string;
}) {
  const LucideComponent = iconMap[name];
  if (!LucideComponent) return null;

  return <LucideComponent size={size} className={className} aria-hidden />;
}
