import { Cpu, Edit, HardDriveIcon, MemoryStick, Network, Server, Telescope, TerminalSquare } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { Button } from "./ui/button";




export interface NodeCardProps {
  id: number;
  name: string;
  ip: string;
  os: string;
  platform: string;
  platform_version: string;
  kernel_version: string;
  cpus: number;
  total_memory: number;
}


export  function NodeCard(props: NodeCardProps) {

  return <>
    <Card>
      <CardHeader>
        <CardTitle className="grid gap-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xl">
              <Server></Server> {props.name}
            </div>
            <div className="flex items-center gap-2">
              <Button size="icon" variant="outline">
                <Edit></Edit>
              </Button>
              <Button size="icon" variant="outline">
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
            <span className="font-semibold">{props.platform} {props.platform_version} ({props.os})</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <TerminalSquare className="w-5 h-5 text-muted-foreground" />
              <span>Kernel Version</span>
            </div>
            <span className="font-semibold">{props.kernel_version}</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Cpu className="w-5 h-5 text-muted-foreground" />
              <span>CPUs</span>
            </div>
            <span className="font-semibold">{props.cpus} </span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <MemoryStick className="w-5 h-5 text-muted-foreground" />
              <span>Memory</span>
            </div>
            <span className="font-semibold">{props.total_memory.toFixed(2)} GB</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Network className="w-5 h-5 text-muted-foreground" />
              <span>IP Address</span>
            </div>
            <span className="font-semibold">{props.ip}</span>
          </div>

        </div>
      </CardContent>
      {/* <CardFooter>
        <p>Card Footer</p>
      </CardFooter> */}
    </Card>

  </>

}