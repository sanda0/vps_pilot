import { Cpu, Edit, HardDriveIcon, MemoryStick, Network, Server, Telescope, TerminalSquare } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../../components/ui/card";
import { Button } from "../../../components/ui/button";
import { NodeSysInfo } from "@/types/node_type";




export interface NodeCardProps {
  data: NodeSysInfo;
  onEdit?: (id: number,name:string) => void;
  onView?: (id: number) => void;
}




export function NodeCard(props: NodeCardProps) {

  return <>
    <Card>
      <CardHeader>
        <CardTitle className="grid gap-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xl">
              <Server></Server> {props.data.name}
            </div>
            <div className="flex items-center gap-2">
              <Button size="icon" variant="outline" onClick={()=>props.onEdit?.(props.data.id,props.data.name)}>
                <Edit></Edit>
              </Button>
              <Button size="icon" variant="outline" onClick={()=>props.onView?.(props.data.id)}>
                <Telescope></Telescope>
              </Button>
            </div>
          </div>


        </CardTitle>
        <CardDescription>System information</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <HardDriveIcon className="w-5 h-5 text-muted-foreground" />
              <span>OS</span>
            </div>
            <span className="font-semibold">{props.data.platform} {props.data.platform_version} ({props.data.os})</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <TerminalSquare className="w-5 h-5 text-muted-foreground" />
              <span>Kernel Version</span>
            </div>
            <span className="font-semibold">{props.data.kernel_version}</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Cpu className="w-5 h-5 text-muted-foreground" />
              <span>CPUs</span>
            </div>
            <span className="font-semibold">{props.data.cpus} </span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <MemoryStick className="w-5 h-5 text-muted-foreground" />
              <span>Memory</span>
            </div>
            <span className="font-semibold">{props.data.total_memory.toFixed(2)} GB</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Network className="w-5 h-5 text-muted-foreground" />
              <span>IP Address</span>
            </div>
            <span className="font-semibold">{props.data.ip}</span>
          </div>

        </div>
      </CardContent>
      {/* <CardFooter>
        <p>Card Footer</p>
      </CardFooter> */}
    </Card>

  </>

}