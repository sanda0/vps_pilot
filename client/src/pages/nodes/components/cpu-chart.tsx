


import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"


import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { useEffect, useState } from "react"





interface ChartProps {
  timeRange: string
  data: { [key: string]: { time: string; value: number }[] }
  cpuCount: number
}





export function CpuChart(props: ChartProps) {
  const [chartConfig, setChartConfig] = useState<ChartConfig>({})
  const [chartData, setChartData] = useState<any>([])
  useEffect(() => {

    // console.log("props.data", props.data)
    // console.log("props.timeRange", props.timeRange)
    // console.log("props.cpuCount", props.cpuCount);

    const newChartConfig: ChartConfig = {};
    const colorGap = Math.floor(40 / props.cpuCount);
    for (let i = 1; i <= props.cpuCount; i++) {
      newChartConfig[`cpu_${i}`] = {
        label: `CPU ${i}`,
        color: `hsl(var(--chart-${i * colorGap}))`,
      }
    }

    setChartConfig(newChartConfig);
    console.log("data", props.data);
    setChartData(props.data)

  }, [props.data, props.timeRange, props.cpuCount])

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
          dataKey="time"
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => new Date(value).toLocaleString('en-US', { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })}
        />
        <YAxis
          tickLine={false}
          axisLine={false}
          tickFormatter={(value) => `${value}%`}
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

        {Object.keys(chartConfig).map((key) => (
          <Area
            key={key}
            dataKey={key}
            type="natural"
            fill={chartConfig[key].color}
            fillOpacity={0}
            stroke={chartConfig[key].color}
            stackId={key}
          />
        ))}


      </AreaChart>
    </ChartContainer>
  )
}
