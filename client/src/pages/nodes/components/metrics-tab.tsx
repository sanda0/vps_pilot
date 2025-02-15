import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import ButtonBar from "./button-bar";
import { CpuChart } from "./cpu-chart";
import { useEffect, useState } from "react";
import { MemoryChart } from "./memory-chart";
import { NetworkChart } from "./netwrok-chart";
import { useParams } from "react-router";
import { NodeData } from "@/types/node_type";
import api from "@/lib/api";

export default function MetricsTab() {

  const [currentTimeRange, setCurrentTimeRange] = useState<string>("5M")
  const { id } = useParams<{ id: string }>();
  const [memData, setMemData] = useState([]);
  const [cpuData, setCpuData] = useState<any>();
  const [node, setNode] = useState<NodeData | null>(null);

  useEffect(() => {

    api.get(`/nodes/${id}`).then((res) => {
      setNode(res.data.data)
    }).catch((err) => {
      console.error(err)
    })

    const ws = new WebSocket(`ws://localhost:8000/api/v1/nodes/ws/system-stat`);
    let timer:any = null;
  
    ws.onopen = () => {
      console.log('WebSocket connection opened');
      ws.send(JSON.stringify({ id: Number(id), time_range: currentTimeRange }));
  
      // Start timer after WebSocket is open
      timer = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ id: Number(id), time_range: currentTimeRange }));
        }
      }, 10000);
    };
  
    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);

      setMemData(message.mem);
      setCpuData(message.cpu);
    };
  
    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  
    ws.onclose = () => {
      console.log('WebSocket connection closed');
      clearInterval(timer); // Clear timer when connection closes
    };
  
    return () => {
      ws.close();
      clearInterval(timer);
    };
  }, [id, currentTimeRange]);

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
                {cpuData && <CpuChart timeRange={currentTimeRange} cpuCount={node?.cpus??0} data={cpuData}></CpuChart>}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Memory usage</CardTitle>
            </CardHeader>
            <CardContent>
            <MemoryChart timeRange={currentTimeRange} data={memData}></MemoryChart>
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