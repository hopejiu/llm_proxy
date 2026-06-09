import { NavLink, Outlet } from "react-router-dom";
import {
  Server,
  BarChart3,
  ScrollText,
  Activity,
  Settings,
  Circle,
} from "lucide-react";
import { useAppContext } from "../context/AppContext";
import { cn } from "../lib/utils";

const navItems = [
  { to: "/providers", icon: Server, label: "Providers" },
  { to: "/stats", icon: BarChart3, label: "统计" },
  { to: "/logs", icon: ScrollText, label: "日志" },
  { to: "/realtime", icon: Activity, label: "实时" },
  { to: "/settings", icon: Settings, label: "设置" },
];

export default function Layout() {
  const { proxyStatus } = useAppContext();
  const isRunning = proxyStatus.status === "running";

  return (
    <div className="flex h-screen bg-[#1a1a2e] text-gray-200">
      {/* 侧边栏 */}
      <aside className="w-56 bg-[#16213e] border-r border-gray-700/50 flex flex-col">
        <div className="p-4 border-b border-gray-700/50">
          <h1 className="text-lg font-bold text-white">LLM Proxy</h1>
          <div className="flex items-center gap-1.5 mt-2 text-xs">
            <Circle
              size={8}
              className={cn(
                isRunning ? "fill-green-400 text-green-400" : "fill-red-400 text-red-400"
              )}
            />
            <span className="text-gray-400">
              {isRunning ? `:${proxyStatus.port}` : "已停止"}
            </span>
          </div>
        </div>

        <nav className="flex-1 p-2 space-y-1">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors",
                  isActive
                    ? "bg-blue-600/20 text-blue-300"
                    : "text-gray-400 hover:text-gray-200 hover:bg-gray-700/30"
                )
              }
            >
              <Icon size={16} />
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="p-3 border-t border-gray-700/50 text-[11px] text-gray-500">
          v{APP_VERSION}
        </div>
      </aside>

      {/* 主内容 */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}

declare const APP_VERSION: string;
