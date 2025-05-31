


import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"


import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"


const chartConfig = {
  recv: {
    label: "Received",
    color: "hsl(var(--chart-1))",
  },
  sent: {
    label: "Sent",
    color: "hsl(var(--chart-30))",
  },
} satisfies ChartConfig

interface ChartProps {
  timeRange: string
  data: Object[]
}

export function NetworkChart(props: ChartProps) {
  return (

    <ChartContainer config={chartConfig} className="w-full h-[250px]">
      <AreaChart
        accessibilityLayer
        data={props.data}
        margin={{
          left: 20,
          right: 0,
        }}
      >
        <CartesianGrid vertical={true} />
        <XAxis
          dataKey="time"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          tickFormatter={(value) => new Date(value).toLocaleString('en-US', { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })}
        />
        <YAxis
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `${value} B/s`}

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
          dataKey="recv"
          type="natural"
          fill="var(--color-recv)"
          fillOpacity={0.1}
          stroke="var(--color-recv)"
          stackId="a"
        />
        <Area
          dataKey="sent"
          type="natural"
          fill="var(--color-sent)"
          fillOpacity={0.1}
          stroke="var(--color-sent)"
          stackId="b"

        />
      </AreaChart>
    </ChartContainer>
  )
}
