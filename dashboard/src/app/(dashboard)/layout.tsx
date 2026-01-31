"use client";

import { useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { useAuthStore, useHydration } from "@/lib/store";
import { cn } from "@/lib/utils";
import { WebSocketProvider, useWebSocket } from "@/lib/websocket";

const navigation = [
  { name: "Dashboard", href: "/dashboard", icon: "📊" },
  { name: "Servers", href: "/servers", icon: "🖥️" },
  { name: "Alerts", href: "/alerts", icon: "🔔" },
  { name: "Settings", href: "/settings", icon: "⚙️" },
];

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const { token, user, organization, clearAuth } = useAuthStore();
  const hydrated = useHydration();

  useEffect(() => {
    if (hydrated && !token) {
      router.push("/login");
    }
  }, [token, router, hydrated]);

  // Wait for hydration before rendering
  if (!hydrated) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
      </div>
    );
  }

  if (!token) {
    return null;
  }

  const handleLogout = () => {
    clearAuth();
    router.push("/login");
  };

  return (
    <WebSocketProvider>
      <DashboardContent
        pathname={pathname}
        user={user}
        organization={organization}
        handleLogout={handleLogout}
      >
        {children}
      </DashboardContent>
    </WebSocketProvider>
  );
}

function ConnectionStatus() {
  const { isConnected } = useWebSocket();
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 text-xs">
      <div
        className={cn(
          "w-2 h-2 rounded-full",
          isConnected ? "bg-green-500" : "bg-gray-400"
        )}
      />
      <span className="text-gray-500 dark:text-gray-400">
        {isConnected ? "Live" : "Connecting..."}
      </span>
    </div>
  );
}

function DashboardContent({
  children,
  pathname,
  user,
  organization,
  handleLogout,
}: {
  children: React.ReactNode;
  pathname: string;
  user: { name?: string } | null;
  organization: { name?: string } | null;
  handleLogout: () => void;
}) {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <aside className="fixed inset-y-0 left-0 z-50 w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700">
        <div className="flex flex-col h-full">
          {/* Logo */}
          <div className="flex items-center justify-between h-16 px-6 border-b border-gray-200 dark:border-gray-700">
            <span className="text-xl font-bold text-gray-900 dark:text-white">
              Sentinel
            </span>
            <ConnectionStatus />
          </div>

          {/* Navigation */}
          <nav className="flex-1 px-4 py-4 space-y-1">
            {navigation.map((item) => {
              const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
              return (
                <Link
                  key={item.name}
                  href={item.href}
                  className={cn(
                    "flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors",
                    isActive
                      ? "bg-brand-50 text-brand-700 dark:bg-brand-900/50 dark:text-brand-400"
                      : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                  )}
                >
                  <span className="mr-3">{item.icon}</span>
                  {item.name}
                </Link>
              );
            })}
          </nav>

          {/* User menu */}
          <div className="p-4 border-t border-gray-200 dark:border-gray-700">
            <div className="flex items-center">
              <div className="flex-shrink-0 w-10 h-10 bg-brand-100 dark:bg-brand-900 rounded-full flex items-center justify-center">
                <span className="text-brand-600 dark:text-brand-400 font-medium">
                  {user?.name?.charAt(0).toUpperCase()}
                </span>
              </div>
              <div className="ml-3 flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-white truncate">
                  {user?.name}
                </p>
                <p className="text-xs text-gray-500 truncate">
                  {organization?.name}
                </p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              className="mt-3 w-full text-left px-3 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
            >
              Sign out
            </button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="pl-64">
        <div className="py-8 px-8">{children}</div>
      </main>
    </div>
  );
}

