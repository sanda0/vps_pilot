import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import ButtonBar from "./button-bar";
import { CpuChart } from "./cpu-chart";
import { useState } from "react";
import { MemoryChart } from "./memory-chart";
import { NetworkChart } from "./netwrok-chart";

export default function MetricsTab() {

  const [currentTimeRange, setCurrentTimeRange] = useState<string>("5M")

  return <>
    <div>
      <div className="flex justify-between">
        <div className="text-2xl font-semibold">Matrics</div>
        <div className="flex items-center ">
          <ButtonBar list={["5M", "15M", "1H", "1D", "2D", "7D"]} onClick={(v)=>setCurrentTimeRange(v)} ></ButtonBar>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-4 mt-4">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-1 xl:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>CPU usage</CardTitle>
            </CardHeader>
            <CardContent>
                <CpuChart timeRange={currentTimeRange}></CpuChart>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Memory usage</CardTitle>
            </CardHeader>
            <CardContent>
            <MemoryChart timeRange={currentTimeRange}></MemoryChart>
            </CardContent>
          </Card>
        </div>
        <div className="grid grid-cols-1">
          <Card>
            <CardHeader>
              <CardTitle>Newwork</CardTitle>
            </CardHeader>
            <CardContent>
                <NetworkChart timeRange={currentTimeRange}></NetworkChart>
            </CardContent>
          </Card>
        </div>

      </div>
    </div>
  </>

}