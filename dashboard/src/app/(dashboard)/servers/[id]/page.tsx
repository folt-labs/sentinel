"use client";

import { useState, useMemo } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { serversApi, type SecurityEvent } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { useWebSocket } from "@/lib/websocket";
import { cn, formatDate, formatRelativeTime, getSeverityColor, getStatusColor } from "@/lib/utils";
import { MetricsChart } from "@/components/MetricsChart";
import { DateRangePicker, type DateRange } from "@/components/DateRangePicker";

function EventRow({ event, expanded, onToggle }: { event: SecurityEvent; expanded: boolean; onToggle: () => void }) {
  return (
    <div
      className="p-4 border-b border-gray-100 dark:border-gray-700 last:border-0 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer transition-colors"
      onClick={onToggle}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "px-2 py-0.5 text-xs font-medium rounded",
                getSeverityColor(event.severity)
              )}
            >
              {event.severity}
            </span>
            <span className="text-sm font-medium text-gray-900 dark:text-white">
              {event.event_type.replace(/_/g, " ")}
            </span>
          </div>
          {expanded && (
            <div className="mt-3 space-y-2">
              {typeof event.data.raw_log === 'string' && (
                <div className="bg-gray-50 dark:bg-gray-700 p-3 rounded font-mono text-xs text-gray-600 dark:text-gray-300 overflow-x-auto">
                  {event.data.raw_log}
                </div>
              )}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                {typeof event.data.username === 'string' && (
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">User:</span>{" "}
                    <span className="text-gray-900 dark:text-white">{event.data.username}</span>
                  </div>
                )}
                {typeof event.data.source_ip === 'string' && (
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Source IP:</span>{" "}
                    <span className="text-gray-900 dark:text-white">{event.data.source_ip}</span>
                  </div>
                )}
                {typeof event.data.auth_method === 'string' && (
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Auth:</span>{" "}
                    <span className="text-gray-900 dark:text-white">{event.data.auth_method}</span>
                  </div>
                )}
                {typeof event.data.file === 'string' && (
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">File:</span>{" "}
                    <span className="text-gray-900 dark:text-white font-mono">{event.data.file}</span>
                  </div>
                )}
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">
                {formatDate(event.timestamp)}
              </div>
            </div>
          )}
        </div>
        <div className="flex items-center gap-2 ml-4">
          <span className="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
            {formatRelativeTime(event.timestamp)}
          </span>
          <svg
            className={cn(
              "w-4 h-4 text-gray-400 transition-transform",
              expanded && "rotate-180"
            )}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>
    </div>
  );
}

export default function ServerDetailPage() {
  const params = useParams();
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const queryClient = useQueryClient();
  const { isConnected } = useWebSocket();
  const serverId = params.id as string;

  const [expandedEvent, setExpandedEvent] = useState<string | null>(null);
  const [severityFilter, setSeverityFilter] = useState<string>("all");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [dateRange, setDateRange] = useState<DateRange>({
    startDate: null,
    endDate: null,
    label: "All time",
  });

  // Only poll when WebSocket is disconnected (fallback)
  const pollInterval = isConnected ? false : 30000;

  const { data: server, isLoading: serverLoading } = useQuery({
    queryKey: ["server", serverId],
    queryFn: () => serversApi.get(token!, serverId),
    enabled: !!token && !!serverId,
    refetchInterval: pollInterval,
  });

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ["server-events", serverId, dateRange.startDate, dateRange.endDate],
    queryFn: () =>
      serversApi.getEvents(token!, serverId, {
        limit: 500,
        startDate: dateRange.startDate || undefined,
        endDate: dateRange.endDate || undefined,
      }),
    enabled: !!token && !!serverId,
    refetchInterval: pollInterval,
  });

  const { data: metricsData, isLoading: metricsLoading } = useQuery({
    queryKey: ["server-metrics", serverId],
    queryFn: () => serversApi.getMetrics(token!, serverId),
    enabled: !!token && !!serverId,
    refetchInterval: isConnected ? 60000 : 60000, // Metrics still poll every 60s (they're historical data)
  });

  const events = eventsData?.events || [];
  const metrics = metricsData?.metrics || [];

  // Compute stats
  const stats = useMemo(() => {
    const byType: Record<string, number> = {};
    const bySeverity: Record<string, number> = {};

    events.forEach(e => {
      byType[e.event_type] = (byType[e.event_type] || 0) + 1;
      bySeverity[e.severity] = (bySeverity[e.severity] || 0) + 1;
    });

    return { byType, bySeverity, total: events.length };
  }, [events]);

  // Get unique event types for filter
  const eventTypes = useMemo(() => {
    return [...new Set(events.map(e => e.event_type))].sort();
  }, [events]);

  // Filter events
  const filteredEvents = useMemo(() => {
    return events.filter(event => {
      if (severityFilter !== "all" && event.severity !== severityFilter) return false;
      if (typeFilter !== "all" && event.event_type !== typeFilter) return false;
      if (searchQuery) {
        const searchLower = searchQuery.toLowerCase();
        const matchesType = event.event_type.toLowerCase().includes(searchLower);
        const rawLog = event.data.raw_log;
        const username = event.data.username;
        const sourceIp = event.data.source_ip;
        const matchesRawLog = typeof rawLog === 'string' && rawLog.toLowerCase().includes(searchLower);
        const matchesUser = typeof username === 'string' && username.toLowerCase().includes(searchLower);
        const matchesIP = typeof sourceIp === 'string' && sourceIp.includes(searchLower);
        if (!matchesType && !matchesRawLog && !matchesUser && !matchesIP) return false;
      }
      return true;
    });
  }, [events, severityFilter, typeFilter, searchQuery]);

  const deleteMutation = useMutation({
    mutationFn: () => serversApi.delete(token!, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["servers"] });
      toast.success("Server deleted");
      router.push("/servers");
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to delete server");
    },
  });

  const handleDelete = () => {
    if (confirm("Are you sure you want to delete this server? This action cannot be undone.")) {
      deleteMutation.mutate();
    }
  };

  if (serverLoading) {
    return (
      <div className="animate-pulse space-y-6">
        <div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
        <div className="h-32 bg-gray-200 dark:bg-gray-700 rounded-xl" />
      </div>
    );
  }

  if (!server) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500 dark:text-gray-400">Server not found</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <div
              className={cn(
                "w-4 h-4 rounded-full animate-pulse",
                server.status === "online"
                  ? "bg-green-500"
                  : server.status === "offline"
                  ? "bg-gray-400"
                  : server.status === "warning"
                  ? "bg-yellow-500"
                  : "bg-red-500"
              )}
            />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {server.hostname}
            </h1>
          </div>
          <p className="mt-1 text-gray-500 dark:text-gray-400">
            {server.ip_address || "No IP address"}
          </p>
        </div>
        <button
          onClick={handleDelete}
          disabled={deleteMutation.isPending}
          className="px-4 py-2 text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
        >
          Delete Server
        </button>
      </div>

      {/* Server info */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Server Information
        </h2>
        <dl className="grid grid-cols-2 md:grid-cols-4 gap-6">
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Status</dt>
            <dd className="mt-1">
              <span
                className={cn(
                  "inline-block px-2 py-0.5 text-sm font-medium rounded capitalize",
                  getStatusColor(server.status)
                )}
              >
                {server.status}
              </span>
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Agent Version</dt>
            <dd className="mt-1 text-gray-900 dark:text-white">
              {server.agent_version || "Unknown"}
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Last Seen</dt>
            <dd className="mt-1 text-gray-900 dark:text-white">
              {server.last_seen ? formatRelativeTime(server.last_seen) : "Never"}
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Created</dt>
            <dd className="mt-1 text-gray-900 dark:text-white">
              {formatDate(server.created_at)}
            </dd>
          </div>
        </dl>
      </div>

      {/* Resource Usage Chart */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Resource Usage (Last 24 Hours)
        </h2>
        <MetricsChart data={metrics} isLoading={metricsLoading} />
      </div>

      {/* Event Stats */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm">
          <div className="text-2xl font-bold text-gray-900 dark:text-white">{stats.total}</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">Total Events</div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm">
          <div className="text-2xl font-bold text-red-600">{stats.bySeverity.critical || 0}</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">Critical</div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm">
          <div className="text-2xl font-bold text-orange-600">{stats.bySeverity.high || 0}</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">High</div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm">
          <div className="text-2xl font-bold text-yellow-600">{stats.bySeverity.warning || 0}</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">Warning</div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm">
          <div className="text-2xl font-bold text-blue-600">{stats.bySeverity.info || 0}</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">Info</div>
        </div>
      </div>

      {/* Events */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm overflow-hidden">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700 space-y-4">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Events ({filteredEvents.length}{eventsData?.total ? ` of ${eventsData.total}` : ""})
            </h2>
            <div className="flex flex-wrap items-center gap-3">
              <input
                type="text"
                placeholder="Search events..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white w-48"
              />
              <select
                value={severityFilter}
                onChange={(e) => setSeverityFilter(e.target.value)}
                className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="all">All Severities</option>
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="warning">Warning</option>
                <option value="medium">Medium</option>
                <option value="info">Info</option>
              </select>
              <select
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value)}
                className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="all">All Types</option>
                {eventTypes.map(type => (
                  <option key={type} value={type}>
                    {type.replace(/_/g, " ")}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <DateRangePicker value={dateRange} onChange={setDateRange} />
        </div>
        {eventsLoading ? (
          <div className="p-8 text-center">
            <div className="animate-pulse text-gray-500">Loading events...</div>
          </div>
        ) : filteredEvents.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-gray-500 dark:text-gray-400">
              {events.length === 0 ? "No events yet" : "No events match your filters"}
            </p>
          </div>
        ) : (
          <div className="max-h-[600px] overflow-y-auto">
            {filteredEvents.map((event) => (
              <EventRow
                key={event.id}
                event={event}
                expanded={expandedEvent === event.id}
                onToggle={() => setExpandedEvent(expandedEvent === event.id ? null : event.id)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
