"use client";

import { useMemo } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { format } from "date-fns";
import type { MetricSeries } from "@/lib/api";

interface MetricsChartProps {
  data: MetricSeries[];
  isLoading?: boolean;
}

const METRIC_CONFIG: Record<string, { color: string; label: string }> = {
  cpu_percent: { color: "#3b82f6", label: "CPU" },
  memory_percent: { color: "#22c55e", label: "Memory" },
  disk_percent: { color: "#f59e0b", label: "Disk" },
};

export function MetricsChart({ data, isLoading }: MetricsChartProps) {
  // Merge all series by timestamp
  const chartData = useMemo(() => {
    const timestampMap = new Map<string, Record<string, number>>();

    data.forEach((series) => {
      series.data.forEach((point) => {
        const ts = point.timestamp;
        if (!timestampMap.has(ts)) {
          timestampMap.set(ts, { timestamp: new Date(ts).getTime() });
        }
        const entry = timestampMap.get(ts)!;
        entry[series.metric] = Math.round(point.value * 10) / 10;
      });
    });

    return Array.from(timestampMap.values()).sort(
      (a, b) => a.timestamp - b.timestamp
    );
  }, [data]);

  const formatXAxis = (timestamp: number) => {
    return format(new Date(timestamp), "HH:mm");
  };

  const formatTooltipTime = (timestamp: number) => {
    return format(new Date(timestamp), "MMM d, HH:mm");
  };

  if (isLoading) {
    return (
      <div className="h-64 flex items-center justify-center bg-gray-50 dark:bg-gray-700/50 rounded-lg">
        <div className="text-gray-500 dark:text-gray-400 animate-pulse">
          Loading metrics...
        </div>
      </div>
    );
  }

  if (chartData.length === 0) {
    return (
      <div className="h-64 flex items-center justify-center bg-gray-50 dark:bg-gray-700/50 rounded-lg">
        <div className="text-gray-500 dark:text-gray-400">
          No metrics data available
        </div>
      </div>
    );
  }

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart
          data={chartData}
          margin={{ top: 5, right: 20, left: 0, bottom: 5 }}
        >
          <CartesianGrid
            strokeDasharray="3 3"
            className="stroke-gray-200 dark:stroke-gray-700"
          />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatXAxis}
            className="text-xs"
            stroke="#9ca3af"
            tick={{ fill: "#9ca3af" }}
          />
          <YAxis
            domain={[0, 100]}
            tickFormatter={(value) => `${value}%`}
            className="text-xs"
            stroke="#9ca3af"
            tick={{ fill: "#9ca3af" }}
            width={45}
          />
          <Tooltip
            labelFormatter={formatTooltipTime}
            formatter={(value: number, name: string) => [
              `${value}%`,
              METRIC_CONFIG[name]?.label || name,
            ]}
            contentStyle={{
              backgroundColor: "rgba(255, 255, 255, 0.95)",
              border: "1px solid #e5e7eb",
              borderRadius: "0.5rem",
              boxShadow: "0 4px 6px -1px rgb(0 0 0 / 0.1)",
            }}
          />
          <Legend
            formatter={(value) => METRIC_CONFIG[value]?.label || value}
          />
          {data.map((series) => (
            <Line
              key={series.metric}
              type="monotone"
              dataKey={series.metric}
              stroke={METRIC_CONFIG[series.metric]?.color || "#6b7280"}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
