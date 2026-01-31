"use client";

import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { serversApi, type SecurityEvent } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { cn, formatDate, formatRelativeTime, getSeverityColor, getStatusColor } from "@/lib/utils";

function EventRow({ event }: { event: SecurityEvent }) {
  return (
    <div className="p-4 border-b border-gray-100 dark:border-gray-700 last:border-0">
      <div className="flex items-start justify-between">
        <div>
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
          {typeof event.data.raw_log === 'string' && (
            <p className="mt-2 text-sm text-gray-600 dark:text-gray-400 font-mono bg-gray-50 dark:bg-gray-700 p-2 rounded">
              {event.data.raw_log.slice(0, 200)}
              {event.data.raw_log.length > 200 && "..."}
            </p>
          )}
        </div>
        <span className="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
          {formatRelativeTime(event.timestamp)}
        </span>
      </div>
    </div>
  );
}

export default function ServerDetailPage() {
  const params = useParams();
  const router = useRouter();
  const token = useAuthStore((s) => s.token);
  const queryClient = useQueryClient();
  const serverId = params.id as string;

  const { data: server, isLoading: serverLoading } = useQuery({
    queryKey: ["server", serverId],
    queryFn: () => serversApi.get(token!, serverId),
    enabled: !!token && !!serverId,
  });

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ["server-events", serverId],
    queryFn: () => serversApi.getEvents(token!, serverId),
    enabled: !!token && !!serverId,
    refetchInterval: 30000,
  });

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
                "w-4 h-4 rounded-full",
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

      {/* Recent events */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm overflow-hidden">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            Recent Events
          </h2>
        </div>
        {eventsLoading ? (
          <div className="p-8 text-center">
            <div className="animate-pulse text-gray-500">Loading events...</div>
          </div>
        ) : eventsData?.events?.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-gray-500 dark:text-gray-400">No events yet</p>
          </div>
        ) : (
          <div className="max-h-96 overflow-y-auto">
            {eventsData?.events?.map((event) => (
              <EventRow key={event.id} event={event} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
