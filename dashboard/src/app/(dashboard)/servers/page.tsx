"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { toast } from "sonner";
import { serversApi, type Server } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { cn, formatRelativeTime, getStatusColor } from "@/lib/utils";

function AddServerModal({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const token = useAuthStore((s) => s.token);
  const queryClient = useQueryClient();
  const [hostname, setHostname] = useState("");
  const [ipAddress, setIpAddress] = useState("");
  const [apiKey, setApiKey] = useState<string | null>(null);

  const createMutation = useMutation({
    mutationFn: () =>
      serversApi.create(token!, { hostname, ip_address: ipAddress || undefined }),
    onSuccess: (data) => {
      setApiKey(data.api_key);
      queryClient.invalidateQueries({ queryKey: ["servers"] });
      toast.success("Server created successfully");
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to create server");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createMutation.mutate();
  };

  const handleClose = () => {
    setHostname("");
    setIpAddress("");
    setApiKey(null);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
        {apiKey ? (
          <>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              Server Created
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Copy this API key and use it to configure your agent. This is the only time you will see this key.
            </p>
            <div className="p-4 bg-gray-100 dark:bg-gray-700 rounded-lg font-mono text-sm break-all">
              {apiKey}
            </div>
            <button
              onClick={() => {
                navigator.clipboard.writeText(apiKey);
                toast.success("API key copied to clipboard");
              }}
              className="mt-4 w-full py-2 px-4 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors"
            >
              Copy API Key
            </button>
            <button
              onClick={handleClose}
              className="mt-2 w-full py-2 px-4 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
            >
              Close
            </button>
          </>
        ) : (
          <>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              Add Server
            </h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Hostname
                </label>
                <input
                  type="text"
                  required
                  value={hostname}
                  onChange={(e) => setHostname(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700"
                  placeholder="server.example.com"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  IP Address (optional)
                </label>
                <input
                  type="text"
                  value={ipAddress}
                  onChange={(e) => setIpAddress(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700"
                  placeholder="192.168.1.100"
                />
              </div>
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={handleClose}
                  className="flex-1 py-2 px-4 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending}
                  className="flex-1 py-2 px-4 bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
                >
                  {createMutation.isPending ? "Creating..." : "Create"}
                </button>
              </div>
            </form>
          </>
        )}
      </div>
    </div>
  );
}

function ServerRow({ server }: { server: Server }) {
  return (
    <Link
      href={`/servers/${server.id}`}
      className="block p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div
            className={cn(
              "w-3 h-3 rounded-full",
              server.status === "online"
                ? "bg-green-500"
                : server.status === "offline"
                ? "bg-gray-400"
                : server.status === "warning"
                ? "bg-yellow-500"
                : "bg-red-500"
            )}
          />
          <div>
            <p className="font-medium text-gray-900 dark:text-white">
              {server.hostname}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {server.ip_address || "No IP"}
            </p>
          </div>
        </div>
        <div className="text-right">
          <span
            className={cn(
              "inline-block px-2 py-0.5 text-xs font-medium rounded capitalize",
              getStatusColor(server.status)
            )}
          >
            {server.status}
          </span>
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {server.last_seen
              ? `Last seen ${formatRelativeTime(server.last_seen)}`
              : "Never connected"}
          </p>
        </div>
      </div>
    </Link>
  );
}

export default function ServersPage() {
  const token = useAuthStore((s) => s.token);
  const [showAddModal, setShowAddModal] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["servers"],
    queryFn: () => serversApi.list(token!),
    enabled: !!token,
    refetchInterval: 30000,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Servers</h1>
          <p className="mt-1 text-gray-500 dark:text-gray-400">
            Manage your monitored servers
          </p>
        </div>
        <button
          onClick={() => setShowAddModal(true)}
          className="px-4 py-2 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors"
        >
          Add Server
        </button>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center">
            <div className="animate-pulse text-gray-500">Loading servers...</div>
          </div>
        ) : data?.servers?.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-gray-500 dark:text-gray-400">No servers yet</p>
            <p className="mt-2 text-sm text-gray-400 dark:text-gray-500">
              Add a server to start monitoring
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {data?.servers?.map((server) => (
              <ServerRow key={server.id} server={server} />
            ))}
          </div>
        )}
      </div>

      <AddServerModal isOpen={showAddModal} onClose={() => setShowAddModal(false)} />
    </div>
  );
}
