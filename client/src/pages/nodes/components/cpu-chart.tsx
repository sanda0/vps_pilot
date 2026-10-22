import { useMemo } from "react"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"

import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { formatPercent, formatTooltipTime, timeAxisProps, tooltipRow, TimePoint } from "./chart-format"

const chartConfig = {
  value: {
    label: "CPU",
    color: "hsl(var(--viz-1))",
  },
} satisfies ChartConfig

interface ChartProps {
  timeRange: string
  // One row per timestamp with a cpu_<n> key per core.
  data: (TimePoint & Record<string, number>)[]
}

// Average across cores, one point per timestamp.
function averageCores(rows: ChartProps["data"]) {
  return rows.map((row) => {
    const cores = Object.keys(row).filter((k) => k.startsWith("cpu_"))
    const sum = cores.reduce((s, k) => s + row[k], 0)
    return { timestamp: row.timestamp, value: cores.length ? sum / cores.length : 0 }
  })
}

export function CpuChart(props: ChartProps) {
  const data = useMemo(() => averageCores(props.data ?? []), [props.data])

  return (
    <ChartContainer config={chartConfig} className="w-full h-64">
      <AreaChart data={data} margin={{ top: 8, left: 0, right: 8 }}>
        <CartesianGrid vertical={false} />
        <XAxis {...timeAxisProps(data, props.timeRange)} />
        <YAxis
          domain={[0, 100]}
          ticks={[0, 25, 50, 75, 100]}
          tickFormatter={(v) => `${v}%`}
          tickLine={false}
          axisLine={false}
          width={40}
        />
        <ChartTooltip
          cursor={{ stroke: "hsl(var(--muted-foreground))", strokeWidth: 1 }}
          content={<ChartTooltipContent
            labelFormatter={(_, payload) => formatTooltipTime(payload[0]?.payload.timestamp)}
            formatter={tooltipRow(formatPercent)}
          />}
        />
        <Area
          dataKey="value"
          name="CPU (avg)"
          type="monotone"
          stroke="var(--color-value)"
          strokeWidth={2}
          fill="var(--color-value)"
          fillOpacity={0.08}
          dot={false}
          isAnimationActive={false}
        />
      </AreaChart>
    </ChartContainer>
  )
}
