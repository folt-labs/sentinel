"use client";

import { useAuthStore } from "@/lib/store";

export default function SettingsPage() {
  const { user, organization } = useAuthStore();

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
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Notification Channels
        </h2>
        <p className="text-gray-500 dark:text-gray-400 mb-4">
          Configure how you receive alerts
        </p>
        <div className="space-y-4">
          <div className="p-4 border border-dashed border-gray-300 dark:border-gray-600 rounded-lg text-center">
            <p className="text-gray-500 dark:text-gray-400">
              Notification channel management coming soon
            </p>
          </div>
        </div>
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
    </div>
  );
}
