const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface RequestOptions extends RequestInit {
  token?: string;
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { token, ...fetchOptions } = options;

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_URL}${endpoint}`, {
    ...fetchOptions,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.error || `Request failed: ${response.status}`);
  }

  return response.json();
}

// Auth API
export const authApi = {
  register: (data: {
    email: string;
    password: string;
    name: string;
    organization_name: string;
  }) =>
    request<{
      token: string;
      user: User;
      organization: Organization;
    }>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  login: (email: string, password: string) =>
    request<{
      token: string;
      user: User;
      organization: Organization;
    }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  getMe: (token: string) =>
    request<User>("/api/v1/me", { token }),
};

// Servers API
export const serversApi = {
  list: (token: string) =>
    request<{ servers: Server[] }>("/api/v1/servers", { token }),

  get: (token: string, id: string) =>
    request<Server>(`/api/v1/servers/${id}`, { token }),

  create: (token: string, data: { hostname: string; ip_address?: string }) =>
    request<{ server: Server; api_key: string }>("/api/v1/servers", {
      method: "POST",
      body: JSON.stringify(data),
      token,
    }),

  delete: (token: string, id: string) =>
    request(`/api/v1/servers/${id}`, {
      method: "DELETE",
      token,
    }),

  getEvents: (token: string, id: string, limit = 100, offset = 0) =>
    request<{ events: SecurityEvent[] }>(
      `/api/v1/servers/${id}/events?limit=${limit}&offset=${offset}`,
      { token }
    ),
};

// Alerts API
export const alertsApi = {
  list: (token: string, status?: string, severity?: string) => {
    const params = new URLSearchParams();
    if (status) params.set("status", status);
    if (severity) params.set("severity", severity);
    const query = params.toString();
    return request<{ alerts: Alert[] }>(
      `/api/v1/alerts${query ? `?${query}` : ""}`,
      { token }
    );
  },

  get: (token: string, id: string) =>
    request<Alert>(`/api/v1/alerts/${id}`, { token }),

  acknowledge: (token: string, id: string) =>
    request(`/api/v1/alerts/${id}/acknowledge`, {
      method: "POST",
      token,
    }),

  resolve: (token: string, id: string) =>
    request(`/api/v1/alerts/${id}/resolve`, {
      method: "POST",
      token,
    }),
};

// Dashboard API
export const dashboardApi = {
  getSummary: (token: string) =>
    request<DashboardSummary>("/api/v1/dashboard/summary", { token }),
};

// Types
export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  organization_id: string;
  created_at: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  created_at: string;
}

export interface Server {
  id: string;
  hostname: string;
  ip_address: string;
  status: "online" | "offline" | "warning" | "critical";
  last_seen: string | null;
  agent_version: string;
  created_at: string;
}

export interface SecurityEvent {
  id: string;
  server_id: string;
  event_type: string;
  severity: string;
  timestamp: string;
  data: Record<string, unknown>;
}

export interface Alert {
  id: string;
  server_id: string;
  severity: string;
  status: "open" | "acknowledged" | "resolved";
  title: string;
  description: string;
  triggered_at: string;
  acknowledged_at: string | null;
  resolved_at: string | null;
}

export interface DashboardSummary {
  total_servers: number;
  online_servers: number;
  open_alerts: number;
  critical_alerts: number;
  events_today: number;
  servers_by_status: Record<string, number>;
}
