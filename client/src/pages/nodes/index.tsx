import { NodeCard } from "@/pages/nodes/components/node-card";

import api from "@/lib/api";
import { Plus, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import { JoinTokenDialog } from "@/components/join-token-dialog";
import { useEffect, useState } from "react";
import { NameEditDialog } from "./components/name-edit-dialog";
import { useNavigate } from "react-router";
import { NodeSysInfo } from "@/types/node_type";



export default function NodesIndex() {

  const [nodes, setNodes] = useState<NodeSysInfo[]>([])
  const [currentEditNodeId, setCurrentEditNodeId] = useState<number | null>(null)
  const [currentEditNodeName, setCurrentEditNodeName] = useState<string | null>(null)
  const [openEditDialog, setOpenEditDialog] = useState(false)
  const [openJoinDialog, setOpenJoinDialog] = useState(false)

  const navigate  = useNavigate()

  const loadNodes = () => {
    api.get("/nodes", {
      params: {
        limit: 50,
        page: 1,
        search: ""
      }
    }).then((res) => {
      if (res.status === 200) {
        setNodes(res.data.data)
      }
    }).catch((err) => {
      console.error(err)
    })
  }

  useEffect(() => {
    loadNodes()
  }, [])

  const handleView = (id: number) => {
    navigate(`/nodes/${id}`)
  }

  return (
    <>
      <NameEditDialog open={openEditDialog} onSaved={loadNodes} onOpenChange={(s) => { setOpenEditDialog(s) }} name={currentEditNodeName ?? undefined} id={currentEditNodeId ?? undefined}></NameEditDialog>
      <JoinTokenDialog open={openJoinDialog} onOpenChange={(s) => { setOpenJoinDialog(s); if (!s) loadNodes() }}></JoinTokenDialog>
      <div className="w-full h-16 mx-auto rounded-xl ">

        <div className="flex items-center justify-between h-full px-4">
          <div className="flex items-center gap-2 text-2xl font-semibold"><Server className="w-6 h-6"></Server> Nodes</div>
          <Button onClick={() => setOpenJoinDialog(true)}><Plus className="w-4 h-4" /> Add node</Button>
        </div>
      </div>
      <div className="w-full h-full mx-auto rounded-xl " >
        <div className="grid grid-cols-1 gap-4 p-4 sm:grid-cols-2 xl:grid-cols-3">
          {nodes.map((node) => (
            <NodeCard key={node.id} data={node} onEdit={(id, name) => {
              setCurrentEditNodeId(id);
              setCurrentEditNodeName(name);
              setOpenEditDialog(true);    
            }}
            onView={handleView}
            ></NodeCard>
          ))}
        </div>

      </div>

    </>
  )
}