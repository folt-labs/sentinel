"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { alertsApi, type Alert } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { cn, formatDate, formatRelativeTime, getSeverityColor } from "@/lib/utils";

function AlertCard({ alert }: { alert: Alert }) {
  const token = useAuthStore((s) => s.token);
  const queryClient = useQueryClient();

  const acknowledgeMutation = useMutation({
    mutationFn: () => alertsApi.acknowledge(token!, alert.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
      toast.success("Alert acknowledged");
    },
  });

  const resolveMutation = useMutation({
    mutationFn: () => alertsApi.resolve(token!, alert.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
      toast.success("Alert resolved");
    },
  });

  return (
    <div className="p-4 bg-white dark:bg-gray-800 rounded-xl shadow-sm">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span
              className={cn(
                "px-2 py-0.5 text-xs font-medium rounded",
                getSeverityColor(alert.severity)
              )}
            >
              {alert.severity}
            </span>
            <span
              className={cn(
                "px-2 py-0.5 text-xs font-medium rounded",
                alert.status === "open"
                  ? "bg-red-100 text-red-700"
                  : alert.status === "acknowledged"
                  ? "bg-yellow-100 text-yellow-700"
                  : "bg-green-100 text-green-700"
              )}
            >
              {alert.status}
            </span>
          </div>
          <h3 className="mt-2 text-lg font-medium text-gray-900 dark:text-white">
            {alert.title}
          </h3>
          {alert.description && (
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
              {alert.description}
            </p>
          )}
          <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
            Triggered {formatRelativeTime(alert.triggered_at)}
          </p>
        </div>
      </div>

      {alert.status !== "resolved" && (
        <div className="mt-4 flex gap-2">
          {alert.status === "open" && (
            <button
              onClick={() => acknowledgeMutation.mutate()}
              disabled={acknowledgeMutation.isPending}
              className="px-3 py-1.5 text-sm bg-yellow-100 text-yellow-700 rounded-lg hover:bg-yellow-200 transition-colors disabled:opacity-50"
            >
              Acknowledge
            </button>
          )}
          <button
            onClick={() => resolveMutation.mutate()}
            disabled={resolveMutation.isPending}
            className="px-3 py-1.5 text-sm bg-green-100 text-green-700 rounded-lg hover:bg-green-200 transition-colors disabled:opacity-50"
          >
            Resolve
          </button>
        </div>
      )}
    </div>
  );
}

export default function AlertsPage() {
  const token = useAuthStore((s) => s.token);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [severityFilter, setSeverityFilter] = useState<string>("");

  const { data, isLoading } = useQuery({
    queryKey: ["alerts", statusFilter, severityFilter],
    queryFn: () => alertsApi.list(token!, statusFilter, severityFilter),
    enabled: !!token,
    refetchInterval: 30000,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Alerts</h1>
        <p className="mt-1 text-gray-500 dark:text-gray-400">
          Security alerts from your monitored servers
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
        >
          <option value="">All Statuses</option>
          <option value="open">Open</option>
          <option value="acknowledged">Acknowledged</option>
          <option value="resolved">Resolved</option>
        </select>

        <select
          value={severityFilter}
          onChange={(e) => setSeverityFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
        >
          <option value="">All Severities</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
          <option value="info">Info</option>
        </select>
      </div>

      {/* Alerts list */}
      {isLoading ? (
        <div className="animate-pulse space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-32 bg-gray-200 dark:bg-gray-700 rounded-xl" />
          ))}
        </div>
      ) : data?.alerts?.length === 0 ? (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-8 text-center shadow-sm">
          <p className="text-gray-500 dark:text-gray-400">No alerts found</p>
          <p className="mt-2 text-sm text-gray-400 dark:text-gray-500">
            {statusFilter || severityFilter
              ? "Try adjusting your filters"
              : "Your servers are running smoothly"}
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {data?.alerts?.map((alert) => (
            <AlertCard key={alert.id} alert={alert} />
          ))}
        </div>
      )}
    </div>
  );
}
