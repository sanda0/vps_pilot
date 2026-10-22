import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"

import {
  ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { formatBytesPerSec, formatTooltipTime, roundTicks, timeAxisProps, tooltipRow, TimePoint } from "./chart-format"

const chartConfig = {
  recv: {
    label: "Received",
    color: "hsl(var(--viz-1))",
  },
  sent: {
    label: "Sent",
    color: "hsl(var(--viz-2))",
  },
} satisfies ChartConfig

interface ChartProps {
  timeRange: string
  data: (TimePoint & { sent: number; recv: number })[]
}

export function NetworkChart(props: ChartProps) {
  const data = props.data ?? []
  const ticks = roundTicks(data.reduce((max, d) => Math.max(max, d.recv, d.sent), 0))

  return (
    <ChartContainer config={chartConfig} className="w-full h-64">
      <AreaChart data={data} margin={{ top: 8, left: 0, right: 8 }}>
        <CartesianGrid vertical={false} />
        <XAxis {...timeAxisProps(data, props.timeRange)} />
        <YAxis
          domain={[0, ticks[ticks.length - 1] || 1]}
          ticks={ticks}
          tickFormatter={formatBytesPerSec}
          tickLine={false}
          axisLine={false}
          width={72}
        />
        <ChartTooltip
          cursor={{ stroke: "hsl(var(--muted-foreground))", strokeWidth: 1 }}
          content={<ChartTooltipContent
            labelFormatter={(_, payload) => formatTooltipTime(payload[0]?.payload.timestamp)}
            formatter={tooltipRow(formatBytesPerSec)}
          />}
        />
        <ChartLegend content={<ChartLegendContent />} />
        {(["recv", "sent"] as const).map((key) => (
          <Area
            key={key}
            dataKey={key}
            name={chartConfig[key].label}
            type="monotone"
            stroke={`var(--color-${key})`}
            strokeWidth={2}
            fill={`var(--color-${key})`}
            fillOpacity={0.06}
            dot={false}
            isAnimationActive={false}
          />
        ))}
      </AreaChart>
    </ChartContainer>
  )
}
