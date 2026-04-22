import {
  LayoutDashboard, Package, Truck, Users, ClipboardCheck, Settings,
  TrendingUp, LogOut, Menu, ChevronRight, Sun, Moon, Monitor, Warehouse,
  ArrowLeft, AlertTriangle, RefreshCw, ChevronLeft, type LucideIcon,
  Boxes, ArrowDownUp, Lightbulb,
} from 'lucide-react';

const iconMap: Record<string, LucideIcon> = {
  dashboard: LayoutDashboard,
  loadingBay: Boxes,
  transfers: ArrowDownUp,
  fleet: Truck,
  staff: Users,
  insights: Lightbulb,
  manifests: ClipboardCheck,
  inventory: Package,
  warehouse: Warehouse,
  settings: Settings,
  analytics: TrendingUp,
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
};

export default function Icon({ name, size = 24, className = '' }: { name: string; size?: number; className?: string }) {
  const LucideComponent = iconMap[name];
  if (!LucideComponent) return null;
  return <LucideComponent size={size} strokeWidth={1.75} className={className} />;
}

export { iconMap };
