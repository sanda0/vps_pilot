// Shared axis and tooltip helpers for the metrics charts. Timestamps are Unix seconds.

export interface TimePoint {
  timestamp: number
}

// Seconds between X-axis ticks for each time range button.
const TICK_STEP: Record<string, number> = {
  "5M": 60,
  "15M": 180,
  "1H": 600,
  "1D": 4 * 3600,
  "2D": 8 * 3600,
  "7D": 24 * 3600,
}

// Ticks on round local times (every minute, every 4 hours, ...) between the first and last point.
function timeTicks(data: TimePoint[], timeRange: string) {
  if (!data.length) return []
  const step = TICK_STEP[timeRange] ?? 600
  const offset = new Date().getTimezoneOffset() * 60
  const ticks = []
  let t = Math.ceil((data[0].timestamp - offset) / step) * step + offset
  for (; t <= data[data.length - 1].timestamp; t += step) ticks.push(t)
  return ticks
}

function tickFormatter(timeRange: string) {
  const opts: Intl.DateTimeFormatOptions =
    timeRange === "7D" ? { month: "short", day: "numeric" }
    : timeRange === "2D" ? { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", hourCycle: "h23" }
    : { hour: "2-digit", minute: "2-digit", hourCycle: "h23" }
  return (ts: number) => new Date(ts * 1000).toLocaleString("en-US", opts)
}

// Props for a time-based XAxis.
export function timeAxisProps(data: TimePoint[], timeRange: string) {
  return {
    dataKey: "timestamp",
    type: "number" as const,
    domain: ["dataMin", "dataMax"],
    ticks: timeTicks(data, timeRange),
    tickFormatter: tickFormatter(timeRange),
    tickLine: false,
    axisLine: false,
    tickMargin: 8,
  }
}

export function formatTooltipTime(ts: number) {
  return new Date(ts * 1000).toLocaleString("en-US", {
    month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23",
  })
}

export function formatPercent(v: number) {
  return `${v.toFixed(1)}%`
}

// Decimal units (1 KB = 1000 B), as network rates are usually shown.
export function formatBytesPerSec(v: number) {
  const units = ["B/s", "KB/s", "MB/s", "GB/s"]
  let i = 0
  while (v >= 1000 && i < units.length - 1) {
    v /= 1000
    i++
  }
  return `${i > 0 && v < 10 ? v.toFixed(1) : Math.round(v)} ${units[i]}`
}

// About four Y ticks on round values (0, 500 KB/s, 1.0 MB/s, ...) up to max.
export function roundTicks(max: number) {
  const raw = max / 4
  const pow = 10 ** Math.floor(Math.log10(raw || 1))
  const step = ([1, 2, 2.5, 5, 10].find((m) => m * pow >= raw) ?? 10) * pow
  return Array.from({ length: Math.ceil(max / step) + 1 }, (_, i) => i * step)
}

// Tooltip row: colour key, series name, formatted value.
export function tooltipRow(format: (v: number) => string) {
  return (value: unknown, name: unknown, item: { color?: string }) => (
    <div className="flex w-full items-center gap-2">
      <div className="h-2.5 w-2.5 shrink-0" style={{ backgroundColor: item.color }} />
      <span className="text-muted-foreground">{String(name)}</span>
      <span className="ml-auto pl-4 font-mono font-medium tabular-nums">{format(Number(value))}</span>
    </div>
  )
}
