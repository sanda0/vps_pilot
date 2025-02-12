


import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"


import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"


const chartConfig = {
  value: {
    label: "Memory",
    color: "hsl(var(--chart-1))",
  },

} satisfies ChartConfig

interface ChartProps {
  timeRange: string
  data: Object[]
}

export function MemoryChart(props: ChartProps) {
  return (

    <ChartContainer config={chartConfig} className="w-full h-80">
      <AreaChart
        accessibilityLayer
        data={props.data}
        margin={{
          left: -20,
          right: 0,
        }}
      >
        <CartesianGrid vertical={true} />
        <XAxis
          dataKey="time"
          tickLine={false}
          axisLine={false}
          // tickMargin={8}
          // tickCount={10}

          tickFormatter={(value) => new Date(value).toLocaleString('en-US', { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })}
        />
        <YAxis
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `${value}%`}   // domain={[0, 100]}
          // tickCount={10}
          // minTickGap={10}
          ticks={[0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100]}
        />
        <ChartTooltip cursor={false} content={<ChartTooltipContent
          labelFormatter={(value) => new Date(value).toLocaleString('en-US', {
            month: 'short',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: "2-digit",
            hour12: false
          })}
        />} />
        <Area
          dataKey="value"
          type="natural"
          fill="var(--color-value)"
          fillOpacity={0.4}
          stroke="var(--color-value)"
          stackId="a"
        />

      </AreaChart>
    </ChartContainer>
  )
}
