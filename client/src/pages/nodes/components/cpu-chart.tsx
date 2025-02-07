


import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"


import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
const chartData = [
  { month: "January", desktop: 34, mobile: 80 },
  { month: "February", desktop: 50, mobile: 60 },
  { month: "March", desktop: 37, mobile: 70 },
  { month: "April", desktop: 73, mobile: 90 },
  { month: "May", desktop: 29, mobile: 30 },
  { month: "June", desktop: 44, mobile: 40 },
]

const chartConfig = {
  desktop: {
    label: "Desktop",
    color: "hsl(var(--chart-1))",
  },
  mobile: {
    label: "Mobile",
    color: "hsl(var(--chart-2))",
  },
} satisfies ChartConfig

interface ChartProps {
  timeRange: string
}

export function CpuChart(props: ChartProps) {
  return (

    <ChartContainer config={chartConfig} className="w-full h-80">
      <AreaChart
        accessibilityLayer
        data={chartData}
        margin={{
          left: -20,
          right: 0,
        }}
      >
        <CartesianGrid vertical={true} />
        <XAxis
          dataKey="month"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          tickFormatter={(value) => value.slice(0, 3)}
        />
        <YAxis
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `${value}%`}
          // domain={[0, 100]}
          // tickCount={10}
          // minTickGap={10}
          ticks={[0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100]}
        />
        <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
        <Area
          dataKey="mobile"
          type="natural"
          fill="var(--color-mobile)"
          fillOpacity={0.4}
          stroke="var(--color-mobile)"
          stackId="a"
        />
        <Area
          dataKey="desktop"
          type="natural"
          fill="var(--color-desktop)"
          fillOpacity={0.4}
          stroke="var(--color-desktop)"
          stackId="b"

        />
      </AreaChart>
    </ChartContainer>
  )
}
