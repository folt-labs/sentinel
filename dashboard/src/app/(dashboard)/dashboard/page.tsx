"use client";

import { useQuery } from "@tanstack/react-query";
import { dashboardApi, alertsApi, type DashboardSummary, type Alert } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { useWebSocket } from "@/lib/websocket";
import { cn, formatRelativeTime, getSeverityColor, getStatusColor } from "@/lib/utils";
import Link from "next/link";

function StatCard({
  title,
  value,
  description,
  color = "brand",
}: {
  title: string;
  value: number | string;
  description?: string;
  color?: "brand" | "green" | "yellow" | "red";
}) {
  const colorClasses = {
    brand: "bg-brand-50 text-brand-700 dark:bg-brand-900/50 dark:text-brand-400",
    green: "bg-green-50 text-green-700 dark:bg-green-900/50 dark:text-green-400",
    yellow: "bg-yellow-50 text-yellow-700 dark:bg-yellow-900/50 dark:text-yellow-400",
    red: "bg-red-50 text-red-700 dark:bg-red-900/50 dark:text-red-400",
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
      <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</p>
      <p className={cn("mt-2 text-3xl font-bold", colorClasses[color])}>{value}</p>
      {description && (
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{description}</p>
      )}
    </div>
  );
}

function RecentAlerts({ alerts }: { alerts: Alert[] }) {
  if (alerts.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500 dark:text-gray-400">
        No recent alerts
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {alerts.slice(0, 5).map((alert) => (
        <Link
          key={alert.id}
          href={`/alerts?id=${alert.id}`}
          className="block p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          <div className="flex items-start justify-between">
            <div>
              <span
                className={cn(
                  "inline-block px-2 py-0.5 text-xs font-medium rounded",
                  getSeverityColor(alert.severity)
                )}
              >
                {alert.severity}
              </span>
              <p className="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                {alert.title}
              </p>
            </div>
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {formatRelativeTime(alert.triggered_at)}
            </span>
          </div>
        </Link>
      ))}
    </div>
  );
}

export default function DashboardPage() {
  const token = useAuthStore((s) => s.token);
  const { isConnected } = useWebSocket();

  // Only poll when WebSocket is disconnected (fallback)
  const pollInterval = isConnected ? false : 30000;

  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ["dashboard-summary"],
    queryFn: () => dashboardApi.getSummary(token!),
    enabled: !!token,
    refetchInterval: pollInterval,
  });

  const { data: alertsData, isLoading: alertsLoading } = useQuery({
    queryKey: ["alerts", "open"],
    queryFn: () => alertsApi.list(token!, "open"),
    enabled: !!token,
    refetchInterval: pollInterval,
  });

  if (summaryLoading || alertsLoading) {
    return (
      <div className="animate-pulse space-y-6">
        <div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-32 bg-gray-200 dark:bg-gray-700 rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
        <p className="mt-1 text-gray-500 dark:text-gray-400">
          Overview of your security monitoring
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Total Servers"
          value={summary?.total_servers ?? 0}
          description={`${summary?.online_servers ?? 0} online`}
          color="brand"
        />
        <StatCard
          title="Online Servers"
          value={summary?.online_servers ?? 0}
          color="green"
        />
        <StatCard
          title="Open Alerts"
          value={summary?.open_alerts ?? 0}
          color={summary?.open_alerts ? "yellow" : "green"}
        />
        <StatCard
          title="Critical Alerts"
          value={summary?.critical_alerts ?? 0}
          color={summary?.critical_alerts ? "red" : "green"}
        />
      </div>

      {/* Server status breakdown */}
      {summary?.servers_by_status && Object.keys(summary.servers_by_status).length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            Servers by Status
          </h2>
          <div className="flex flex-wrap gap-4">
            {Object.entries(summary.servers_by_status).map(([status, count]) => (
              <div
                key={status}
                className={cn(
                  "px-4 py-2 rounded-lg",
                  getStatusColor(status)
                )}
              >
                <span className="font-medium capitalize">{status}</span>
                <span className="ml-2">{count}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Recent alerts */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            Recent Alerts
          </h2>
          <Link
            href="/alerts"
            className="text-sm text-brand-600 hover:text-brand-500 font-medium"
          >
            View all
          </Link>
        </div>
        <RecentAlerts alerts={alertsData?.alerts ?? []} />
      </div>

      {/* Events today */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Events Today
        </h2>
        <p className="text-3xl font-bold text-gray-900 dark:text-white">
          {summary?.events_today ?? 0}
        </p>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Security events collected in the last 24 hours
        </p>
      </div>
    </div>
  );
}
