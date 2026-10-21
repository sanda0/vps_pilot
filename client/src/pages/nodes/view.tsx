

import api, { nodesApi } from '@/lib/api';
import { NodeData } from '@/types/node_type';
import { KeyRound, RotateCcw, Server, ShieldOff } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useParams } from 'react-router';
import AnimateDot from './components/animate-dot';
import { Button } from '@/components/ui/button';
import { TooltipWarper } from '@/components/tooltip-warper';
import { NodeViewTabs } from './components/node-view-tabs';
import { JoinTokenDialog } from '@/components/join-token-dialog';
import { useToast } from '@/hooks/use-toast';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

export default function NodeView() {
  const { id } = useParams<{ id: string }>();
  const [node, setNode] = useState<NodeData | null>(null);
  const [isOnline, setIsOnline] = useState<boolean>(true);
  const [openJoinDialog, setOpenJoinDialog] = useState(false);
  const [openRevokeDialog, setOpenRevokeDialog] = useState(false);
  const { toast } = useToast();

  const revokeKey = () => {
    nodesApi.revokeKey(Number(id)).then(() => {
      toast({ title: 'Agent key revoked', description: 'The agent was disconnected. Re-enroll it to reconnect.' });
    }).catch((err) => {
      toast({ title: 'Error', description: err.response?.data?.error || 'Failed to revoke key', variant: 'destructive' });
    });
  };

  useEffect(() => {

    api.get(`/nodes/${id}`).then((res) => {
      setNode(res.data.data)
    }).catch((err) => {
      console.error(err)
    })

    setIsOnline(true);

  }, [id])

  return <>
    <JoinTokenDialog open={openJoinDialog} onOpenChange={setOpenJoinDialog} nodeId={Number(id)}></JoinTokenDialog>
    <AlertDialog open={openRevokeDialog} onOpenChange={setOpenRevokeDialog}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke agent key?</AlertDialogTitle>
          <AlertDialogDescription>
            The agent on <strong>{node?.name}</strong> is disconnected and can't reconnect until you re-enroll it with a new join token. Its history is kept.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={revokeKey} className="bg-destructive hover:bg-destructive/90">Revoke</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    <div className="w-full h-16 mx-auto rounded-xl ">

      <div className="flex items-center justify-between h-full px-4">
        <div className="flex items-center gap-2 text-2xl font-semibold"><Server className="w-10 h-10"></Server>
          <div className="flex flex-col">
            <div className='flex flex-row'>  {node?.name} <AnimateDot color={isOnline ? "success" : "danger"} size={2}></AnimateDot> </div>

            <span className="font-mono text-sm italic font-normal text-gray-500">{node?.ip}</span>
          </div></div>
        <div className="flex gap-2">
          <TooltipWarper tooltipContent={<p>Re-enroll agent</p>} >
            <Button size={"icon"} variant="outline" onClick={() => setOpenJoinDialog(true)}><KeyRound></KeyRound></Button>
          </TooltipWarper>
          <TooltipWarper tooltipContent={<p>Revoke agent key</p>} >
            <Button size={"icon"} variant="outline" onClick={() => setOpenRevokeDialog(true)}><ShieldOff></ShieldOff></Button>
          </TooltipWarper>
          <TooltipWarper tooltipContent={<p>Reboot</p>} >
            <Button size={"icon"} variant="outline" onClick={() => { }}><RotateCcw></RotateCcw></Button>
          </TooltipWarper>
        </div>

      </div>
    </div>
    <div className="w-full h-full px-4 mx-auto">
      <NodeViewTabs></NodeViewTabs>
    </div>
  </>
}