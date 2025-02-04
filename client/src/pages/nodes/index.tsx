import { NodeCard, NodeCardProps } from "@/components/node-card";
import { Button } from "@/components/ui/button";
import api from "@/lib/api";
import { BadgePlus, Server } from "lucide-react";
import { useEffect, useState } from "react";


export default function NodesIndex() {
  
  const [nodes, setNodes] = useState<NodeCardProps[]>([])

  useEffect(() => {
    api.get("/nodes",{
      params:{
        limit: 50,
        page: 1,
        search:""
      }
    }).then((res) => {
      if (res.status === 200) {
        setNodes(res.data.data)
      }
    }).catch((err) => {
      console.error(err)
    })
  }, [])

  return (
    < >
      <div className="w-full h-16 max-w-5xl mx-auto rounded-xl ">
        <div className="flex items-center justify-between h-full px-4">
          <div className="flex items-center gap-2 text-2xl font-semibold"><Server className="w-6 h-6"></Server> Nodes</div>
          <Button>
            <BadgePlus></BadgePlus>
            Add Node
          </Button>
        </div>
      </div>
      <div className="w-full h-full max-w-5xl mx-auto rounded-xl " >
        <div className="grid grid-cols-2 gap-4 p-4">
          {nodes.map((node) => (
            <NodeCard key={node.id} {...node}></NodeCard>
          ))}
        </div>
      </div>

    </>
  )
}