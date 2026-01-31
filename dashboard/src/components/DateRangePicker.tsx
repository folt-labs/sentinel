"use client";

import { useState, useCallback } from "react";
import { format, subDays, subHours, startOfDay, endOfDay } from "date-fns";

export interface DateRange {
  startDate: string | null;
  endDate: string | null;
  label: string;
}

interface DateRangePickerProps {
  value: DateRange;
  onChange: (range: DateRange) => void;
}

const QUICK_FILTERS = [
  { label: "Last 24 hours", hours: 24 },
  { label: "Last 7 days", days: 7 },
  { label: "Last 30 days", days: 30 },
] as const;

export function DateRangePicker({ value, onChange }: DateRangePickerProps) {
  const [showCustom, setShowCustom] = useState(false);
  const [customStart, setCustomStart] = useState("");
  const [customEnd, setCustomEnd] = useState("");

  const handleQuickFilter = useCallback(
    (filter: (typeof QUICK_FILTERS)[number]) => {
      const now = new Date();
      let startDate: Date;

      if ("hours" in filter) {
        startDate = subHours(now, filter.hours);
      } else {
        startDate = subDays(now, filter.days);
      }

      onChange({
        startDate: startDate.toISOString(),
        endDate: now.toISOString(),
        label: filter.label,
      });
      setShowCustom(false);
    },
    [onChange]
  );

  const handleCustomApply = useCallback(() => {
    if (!customStart || !customEnd) return;

    const start = startOfDay(new Date(customStart));
    const end = endOfDay(new Date(customEnd));

    onChange({
      startDate: start.toISOString(),
      endDate: end.toISOString(),
      label: `${format(start, "MMM d")} - ${format(end, "MMM d")}`,
    });
    setShowCustom(false);
  }, [customStart, customEnd, onChange]);

  const handleClear = useCallback(() => {
    onChange({ startDate: null, endDate: null, label: "All time" });
    setCustomStart("");
    setCustomEnd("");
    setShowCustom(false);
  }, [onChange]);

  return (
    <div className="flex flex-wrap items-center gap-2">
      {/* Quick filter buttons */}
      {QUICK_FILTERS.map((filter) => (
        <button
          key={filter.label}
          onClick={() => handleQuickFilter(filter)}
          className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
            value.label === filter.label
              ? "bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300"
              : "bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
          }`}
        >
          {filter.label}
        </button>
      ))}

      {/* Custom toggle */}
      <button
        onClick={() => setShowCustom(!showCustom)}
        className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
          showCustom ||
          (value.label !== "All time" &&
            !QUICK_FILTERS.some((f) => f.label === value.label))
            ? "bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300"
            : "bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
        }`}
      >
        Custom
      </button>

      {/* Clear button */}
      {value.startDate && (
        <button
          onClick={handleClear}
          className="px-3 py-1.5 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
        >
          Clear
        </button>
      )}

      {/* Custom date inputs */}
      {showCustom && (
        <div className="flex items-center gap-2 ml-2">
          <input
            type="date"
            value={customStart}
            onChange={(e) => setCustomStart(e.target.value)}
            className="px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          />
          <span className="text-gray-500 dark:text-gray-400">to</span>
          <input
            type="date"
            value={customEnd}
            onChange={(e) => setCustomEnd(e.target.value)}
            className="px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          />
          <button
            onClick={handleCustomApply}
            disabled={!customStart || !customEnd}
            className="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            Apply
          </button>
        </div>
      )}

      {/* Selected range display */}
      {value.startDate && !showCustom && (
        <span className="text-sm text-gray-500 dark:text-gray-400 ml-2">
          {value.label}
        </span>
      )}
    </div>
  );
}
