

import api from '@/lib/api';
import { NodeData } from '@/types/node_type';
import {  RotateCcw, Server } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useParams } from 'react-router';
import AnimateDot from './components/animate-dot';
import { Button } from '@/components/ui/button';
import { TooltipWarper } from '@/components/tooltip-warper';
import { NodeViewTabs } from './components/node-view-tabs';

export default function NodeView() {
  const { id } = useParams<{ id: string }>();
  const [node, setNode] = useState<NodeData | null>(null);
  const [isOnline, setIsOnline] = useState<boolean>(true);

  useEffect(() => {

    api.get(`/nodes/${id}`).then((res) => {
      setNode(res.data.data)
    }).catch((err) => {
      console.error(err)
    })

    setIsOnline(true);

  }, [id])

  return <>
    <div className="w-full h-16 mx-auto rounded-xl ">

      <div className="flex items-center justify-between h-full px-4">
        <div className="flex items-center gap-2 text-2xl font-semibold"><Server className="w-10 h-10"></Server>
          <div className="flex flex-col">
            <div className='flex flex-row'>  {node?.name} <AnimateDot color={isOnline ? "success" : "danger"} size={2}></AnimateDot> </div>

            <span className="font-mono text-sm italic font-normal text-gray-500">{node?.ip}</span>
          </div></div>
        <TooltipWarper tooltipContent={<p>Reboot</p>} >
          <Button size={"icon"} variant="outline" onClick={() => { }}><RotateCcw></RotateCcw></Button>
        </TooltipWarper>

      </div>
    </div>
    <div className="w-full h-full px-4 mx-auto">
      <NodeViewTabs></NodeViewTabs>
    </div>
  </>
}