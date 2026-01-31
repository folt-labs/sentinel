"use client";

import { useState, useEffect, useCallback } from "react";
import { useAuthStore } from "@/lib/store";
import { settingsApi, NotificationChannel } from "@/lib/api";
import { toast } from "sonner";

export default function SettingsPage() {
  const { user, organization, token } = useAuthStore();
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [testingId, setTestingId] = useState<string | null>(null);

  const fetchChannels = useCallback(async () => {
    if (!token) return;
    try {
      const data = await settingsApi.listNotificationChannels(token);
      setChannels(data.channels || []);
    } catch {
      toast.error("Failed to load notification channels");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    fetchChannels();
  }, [fetchChannels]);

  const handleDelete = async (id: string) => {
    if (!token) return;
    if (!confirm("Are you sure you want to delete this notification channel?")) return;
    try {
      await settingsApi.deleteNotificationChannel(token, id);
      toast.success("Notification channel deleted");
      fetchChannels();
    } catch {
      toast.error("Failed to delete notification channel");
    }
  };

  const handleToggle = async (channel: NotificationChannel) => {
    if (!token) return;
    try {
      await settingsApi.updateNotificationChannel(token, channel.id, {
        enabled: !channel.enabled,
      });
      toast.success(channel.enabled ? "Channel disabled" : "Channel enabled");
      fetchChannels();
    } catch {
      toast.error("Failed to update notification channel");
    }
  };

  const handleTest = async (id: string) => {
    if (!token) return;
    setTestingId(id);
    try {
      await settingsApi.testNotificationChannel(token, id);
      toast.success("Test notification sent!");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send test notification");
    } finally {
      setTestingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Settings</h1>
        <p className="mt-1 text-gray-500 dark:text-gray-400">
          Manage your account and organization settings
        </p>
      </div>

      {/* Profile section */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Profile
        </h2>
        <dl className="space-y-4">
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Name</dt>
            <dd className="mt-1 text-gray-900 dark:text-white">{user?.name}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Email</dt>
            <dd className="mt-1 text-gray-900 dark:text-white">{user?.email}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Role</dt>
            <dd className="mt-1 text-gray-900 dark:text-white capitalize">{user?.role}</dd>
          </div>
        </dl>
      </div>

      {/* Organization section */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Organization
        </h2>
        <dl className="space-y-4">
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Name</dt>
            <dd className="mt-1 text-gray-900 dark:text-white">{organization?.name}</dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Slug</dt>
            <dd className="mt-1 text-gray-900 dark:text-white">{organization?.slug}</dd>
          </div>
        </dl>
      </div>

      {/* Notification channels section */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Notification Channels
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Configure how you receive alerts
            </p>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
          >
            Add Channel
          </button>
        </div>

        {loading ? (
          <div className="text-center py-8 text-gray-500">Loading...</div>
        ) : channels.length === 0 ? (
          <div className="p-4 border border-dashed border-gray-300 dark:border-gray-600 rounded-lg text-center">
            <p className="text-gray-500 dark:text-gray-400 mb-2">
              No notification channels configured
            </p>
            <button
              onClick={() => setShowAddModal(true)}
              className="text-blue-600 hover:text-blue-700 text-sm font-medium"
            >
              Add your first channel
            </button>
          </div>
        ) : (
          <div className="space-y-3">
            {channels.map((channel) => (
              <div
                key={channel.id}
                className="flex items-center justify-between p-4 border border-gray-200 dark:border-gray-700 rounded-lg"
              >
                <div className="flex items-center gap-4">
                  <div
                    className={`w-10 h-10 rounded-lg flex items-center justify-center ${
                      channel.type === "email"
                        ? "bg-blue-100 dark:bg-blue-900"
                        : "bg-purple-100 dark:bg-purple-900"
                    }`}
                  >
                    {channel.type === "email" ? (
                      <svg className="w-5 h-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                      </svg>
                    ) : (
                      <svg className="w-5 h-5 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                      </svg>
                    )}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900 dark:text-white">
                        {channel.name}
                      </span>
                      <span
                        className={`px-2 py-0.5 text-xs rounded-full ${
                          channel.enabled
                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                            : "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
                        }`}
                      >
                        {channel.enabled ? "Active" : "Disabled"}
                      </span>
                    </div>
                    <span className="text-sm text-gray-500 dark:text-gray-400 capitalize">
                      {channel.type}
                      {channel.type === "email" && typeof channel.config.email === 'string' && (
                        <span className="ml-1">- {channel.config.email}</span>
                      )}
                      {channel.type === "webhook" && typeof channel.config.url === 'string' && (
                        <span className="ml-1">- {channel.config.url.substring(0, 30)}...</span>
                      )}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleTest(channel.id)}
                    disabled={testingId === channel.id}
                    className="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors disabled:opacity-50"
                  >
                    {testingId === channel.id ? "Sending..." : "Test"}
                  </button>
                  <button
                    onClick={() => handleToggle(channel)}
                    className="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                  >
                    {channel.enabled ? "Disable" : "Enable"}
                  </button>
                  <button
                    onClick={() => handleDelete(channel.id)}
                    className="px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Agent installation */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Agent Installation
        </h2>
        <p className="text-gray-500 dark:text-gray-400 mb-4">
          Install the Sentinel agent on your Linux servers
        </p>
        <div className="p-4 bg-gray-900 rounded-lg">
          <code className="text-green-400 text-sm font-mono break-all">
            curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- YOUR_API_KEY YOUR_API_URL
          </code>
        </div>
        <p className="mt-4 text-sm text-gray-500 dark:text-gray-400">
          Get your API key when adding a new server from the Servers page.
        </p>
      </div>

      {/* Add Channel Modal */}
      {showAddModal && (
        <AddChannelModal
          onClose={() => setShowAddModal(false)}
          onSuccess={() => {
            setShowAddModal(false);
            fetchChannels();
          }}
        />
      )}
    </div>
  );
}

function AddChannelModal({
  onClose,
  onSuccess,
}: {
  onClose: () => void;
  onSuccess: () => void;
}) {
  const { token } = useAuthStore();
  const [type, setType] = useState<"email" | "webhook">("email");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [webhookUrl, setWebhookUrl] = useState("");
  const [webhookSecret, setWebhookSecret] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;

    setSubmitting(true);
    try {
      const config: Record<string, unknown> =
        type === "email"
          ? { email, recipients: [email] }
          : { url: webhookUrl, secret: webhookSecret || undefined };

      await settingsApi.createNotificationChannel(token, {
        name,
        type,
        config,
      });
      toast.success("Notification channel created");
      onSuccess();
    } catch {
      toast.error("Failed to create notification channel");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md mx-4">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Add Notification Channel
        </h3>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Channel Type
            </label>
            <div className="flex gap-4">
              <label className="flex items-center">
                <input
                  type="radio"
                  name="type"
                  value="email"
                  checked={type === "email"}
                  onChange={() => setType("email")}
                  className="mr-2"
                />
                <span className="text-gray-900 dark:text-white">Email</span>
              </label>
              <label className="flex items-center">
                <input
                  type="radio"
                  name="type"
                  value="webhook"
                  checked={type === "webhook"}
                  onChange={() => setType("webhook")}
                  className="mr-2"
                />
                <span className="text-gray-900 dark:text-white">Webhook</span>
              </label>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Channel Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={type === "email" ? "e.g., Admin Email" : "e.g., Slack Webhook"}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              required
            />
          </div>

          {type === "email" ? (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Email Address
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="alerts@example.com"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                required
              />
            </div>
          ) : (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Webhook URL
                </label>
                <input
                  type="url"
                  value={webhookUrl}
                  onChange={(e) => setWebhookUrl(e.target.value)}
                  placeholder="https://hooks.slack.com/..."
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Secret (optional)
                </label>
                <input
                  type="text"
                  value={webhookSecret}
                  onChange={(e) => setWebhookSecret(e.target.value)}
                  placeholder="For HMAC signature verification"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
                <p className="mt-1 text-xs text-gray-500">
                  If provided, requests will include X-Sentinel-Signature header
                </p>
              </div>
            </>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {submitting ? "Creating..." : "Create Channel"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
