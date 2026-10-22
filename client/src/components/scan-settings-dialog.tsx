import { useCallback, useEffect, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { RefreshCw } from "lucide-react";
import { apiErrorMessage, projectScanApi } from "@/lib/api";
import { ProjectScan } from "@/types/project_scan";
import { useToast } from "@/hooks/use-toast";
import { NodeSelect } from "@/components/node-select";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Scan intervals offered in the dashboard, in seconds (the hub accepts 60..86400).
const INTERVALS: [number, string][] = [
  [60, "Every minute"],
  [300, "Every 5 minutes"],
  [900, "Every 15 minutes"],
  [1800, "Every 30 minutes"],
  [3600, "Every hour"],
  [21600, "Every 6 hours"],
  [86400, "Once a day"],
];

interface ScanSettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// Per-node project search settings: auto scan on/off, interval, rescan now.
export function ScanSettingsDialog({ open, onOpenChange }: ScanSettingsDialogProps) {
  const [nodeId, setNodeId] = useState<number | null>(null);
  const [scan, setScan] = useState<ProjectScan | null>(null);
  const [autoScan, setAutoScan] = useState(true);
  const [scanInterval, setScanInterval] = useState(900);
  const [busy, setBusy] = useState(false);
  const { toast } = useToast();

  const load = useCallback(async (id: number) => {
    try {
      const s = await projectScanApi.get(id);
      setScan(s);
      setAutoScan(s.auto_scan);
      setScanInterval(s.scan_interval);
    } catch (err) {
      setScan(null);
      toast({ variant: "destructive", title: "Error", description: apiErrorMessage(err, "Failed to load scan settings") });
    }
  }, [toast]);

  useEffect(() => {
    if (open && nodeId !== null) load(nodeId);
  }, [open, nodeId, load]);

  const run = async (action: () => Promise<void>, success: string) => {
    setBusy(true);
    try {
      await action();
      toast({ title: success });
      if (nodeId !== null) await load(nodeId);
    } catch (err) {
      toast({ variant: "destructive", title: "Error", description: apiErrorMessage(err, "Request failed") });
    } finally {
      setBusy(false);
    }
  };

  const save = () =>
    run(async () => {
      await projectScanApi.update(nodeId!, autoScan, scanInterval);
    }, scan?.online ? "Saved and sent to the agent" : "Saved; the agent gets it when it reconnects");

  const rescan = () =>
    run(() => projectScanApi.rescan(nodeId!), "Rescan requested; results appear in a few seconds");

  const intervals = INTERVALS.some(([s]) => s === scanInterval) ? INTERVALS : [...INTERVALS, [scanInterval, `Every ${scanInterval} seconds`] as [number, string]];
  const changed = scan !== null && (scan.auto_scan !== autoScan || scan.scan_interval !== scanInterval);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>Scan settings</DialogTitle>
          <DialogDescription>
            How each node's agent searches for <code className="text-xs bg-muted px-1 py-0.5">config.vpspilot.json</code>.
            It always searches when it connects and when you click Rescan.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-5 py-2">
          <div className="grid gap-2">
            <Label>Node</Label>
            <NodeSelect value={nodeId} onChange={setNodeId} />
          </div>

          {scan && (
            <>
              <div className="flex items-center justify-between gap-4 text-sm">
                <div className="flex items-center gap-2">
                  <Badge variant={scan.online ? "default" : "secondary"}>{scan.online ? "Online" : "Offline"}</Badge>
                  <span className="text-muted-foreground">
                    {scan.last_scanned_at
                      ? `Last scanned ${formatDistanceToNow(new Date(scan.last_scanned_at * 1000), { addSuffix: true })}`
                      : "Not scanned yet"}
                  </span>
                </div>
                <Button variant="outline" size="sm" onClick={rescan} disabled={busy || !scan.online}>
                  <RefreshCw className="h-4 w-4" />
                  Rescan now
                </Button>
              </div>

              <div className="flex items-center justify-between gap-4">
                <div>
                  <Label htmlFor="auto-scan">Automatic scan</Label>
                  <p className="text-sm text-muted-foreground">Search again on a schedule to pick up new or changed projects.</p>
                </div>
                <Switch id="auto-scan" checked={autoScan} onCheckedChange={setAutoScan} />
              </div>

              <div className="grid gap-2">
                <Label>Interval</Label>
                <Select value={String(scanInterval)} onValueChange={(v) => setScanInterval(Number(v))} disabled={!autoScan}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {intervals.map(([s, label]) => (
                      <SelectItem key={s} value={String(s)}>{label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Close</Button>
          <Button onClick={save} disabled={busy || !changed}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
