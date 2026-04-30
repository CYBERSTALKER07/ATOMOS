import {
  LayoutDashboard, Package, Truck, Users, ClipboardCheck,
  TrendingUp, LogOut, Sun, Moon, Monitor, Warehouse, Lock,
  AlertTriangle, RefreshCw, type LucideIcon,
  ArrowDownUp, Lightbulb, Plus, ChevronRight, ChevronLeft,
  FileText, Send, CheckCircle, XCircle, Clock, BarChart3,
  ShoppingCart, Boxes, PackageOpen, CreditCard, DollarSign,
  UserCheck, Route,
} from 'lucide-react';

const iconMap: Record<string, LucideIcon> = {
  dashboard: LayoutDashboard,
  supplyRequests: FileText,
  forecast: BarChart3,
  lock: Lock,
  staff: Users,
  fleet: Truck,
  manifests: ClipboardCheck,
  inventory: Package,
  warehouse: Warehouse,
  analytics: TrendingUp,
  logout: LogOut,
  lightMode: Sun,
  darkMode: Moon,
  autoMode: Monitor,
  refresh: RefreshCw,
  warning: AlertTriangle,
  transfers: ArrowDownUp,
  insights: Lightbulb,
  plus: Plus,
  chevronR: ChevronRight,
  left: ChevronLeft,
  right: ChevronRight,
  send: Send,
  check: CheckCircle,
  cancel: XCircle,
  clock: Clock,
  orders: ShoppingCart,
  dispatch: Route,
  catalog: Boxes,
  returns: PackageOpen,
  payment: CreditCard,
  treasury: DollarSign,
  crm: UserCheck,
};

export default function Icon({ name, size = 24, className = '' }: { name: string; size?: number; className?: string }) {
  const LucideComponent = iconMap[name];
  if (!LucideComponent) return null;
  return <LucideComponent size={size} strokeWidth={1.75} className={className} />;
}

export { iconMap };
