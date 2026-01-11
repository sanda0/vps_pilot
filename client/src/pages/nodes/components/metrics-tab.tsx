import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import ButtonBar from "./button-bar";
import { CpuChart } from "./cpu-chart";
import { useEffect, useState, useRef } from "react";
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
  const [networkData, setNetworkData] = useState<any>();
  const wsRef = useRef<WebSocket | null>(null);
  const timerRef = useRef<any>(null);
  const timeRangeRef = useRef<string>(currentTimeRange);

  // Keep timeRangeRef in sync with currentTimeRange
  useEffect(() => {
    timeRangeRef.current = currentTimeRange;
  }, [currentTimeRange]);

  // Fetch node info once
  useEffect(() => {
    api.get(`/nodes/${id}`).then((res) => {
      setNode(res.data.data)
    }).catch((err) => {
      console.error(err)
    })
  }, [id]);

  // WebSocket connection - create once per node
  useEffect(() => {
    const ws = new WebSocket(`ws://localhost:8000/api/v1/nodes/ws/system-stat`);
    wsRef.current = ws;
  
    ws.onopen = () => {
      console.log('WebSocket connection opened');
      // Send initial query
      ws.send(JSON.stringify({ id: Number(id), time_range: timeRangeRef.current }));
  
      // Start timer after WebSocket is open
      timerRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          // Use ref to always get the latest time range
          ws.send(JSON.stringify({ id: Number(id), time_range: timeRangeRef.current }));
        }
      }, 10000);
    };
  
    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      setMemData(message.mem);
      setCpuData(message.cpu);
      setNetworkData(message.net);
    };
  
    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  
    ws.onclose = () => {
      console.log('WebSocket connection closed');
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  
    return () => {
      console.log('Cleaning up WebSocket');
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close();
      }
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [id]); // Only reconnect if node ID changes

  // Send new query immediately when time range changes
  useEffect(() => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      console.log('Time range changed to:', currentTimeRange);
      wsRef.current.send(JSON.stringify({ id: Number(id), time_range: currentTimeRange }));
    }
  }, [currentTimeRange, id]);

  return <>
    <div>
      <div className="flex justify-between">
        <div className="text-2xl font-semibold">Metrics</div>
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
              <CardTitle>Network</CardTitle>
            </CardHeader>
            <CardContent>
                <NetworkChart timeRange={currentTimeRange} data={networkData}></NetworkChart>
            </CardContent>
          </Card>
        </div>

      </div>
    </div>
  </>

}