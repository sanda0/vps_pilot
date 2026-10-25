import { useEffect, useRef, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { Loader2, Square } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { useCommandRun } from "@/hooks/use-command-run";
import { apiErrorMessage, commandRunsApi } from "@/lib/api";
import { cn, formatDuration } from "@/lib/utils";
import { CommandRun } from "@/types/command_run";

export function RunStatusBadge({ run }: { run: CommandRun }) {
  if (run.status === "running") {
    return (
      <Badge variant="secondary" className="gap-1">
        <Loader2 className="h-3 w-3 animate-spin" />
        Running
      </Badge>
    );
  }
  if (run.status === "exited") {
    return run.exit_code === 0 ? (
      <Badge variant="outline">Exit 0</Badge>
    ) : (
      <Badge variant="destructive">Exit {run.exit_code}</Badge>
    );
  }
  const label = run.status.charAt(0).toUpperCase() + run.status.slice(1);
  return <Badge variant={run.status === "cancelled" ? "outline" : "destructive"}>{label}</Badge>;
}

interface CommandRunPanelProps {
  runId: string;
  // Called once the run has finished, e.g. to refresh the run history.
  onFinished?: () => void;
}

export function CommandRunPanel({ runId, onFinished }: CommandRunPanelProps) {
  const { run, output } = useCommandRun(runId);
  const [cancelling, setCancelling] = useState(false);
  const { toast } = useToast();
  const outputRef = useRef<HTMLPreElement>(null);
  const followRef = useRef(true);
  const finished = run !== null && run.status !== "running";

  useEffect(() => {
    if (finished) onFinished?.();
  }, [finished, onFinished]);

  // Keep the newest output in view unless the user scrolled up.
  useEffect(() => {
    const el = outputRef.current;
    if (el && followRef.current) el.scrollTop = el.scrollHeight;
  }, [output]);

  const cancel = async () => {
    setCancelling(true);
    try {
      await commandRunsApi.cancel(runId);
    } catch (err) {
      toast({ variant: "destructive", title: "Couldn't cancel", description: apiErrorMessage(err, "Cancel failed") });
    } finally {
      setCancelling(false);
    }
  };

  if (!run) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{run.command}</span>
          <RunStatusBadge run={run} />
          <span className="text-xs text-muted-foreground">
            by {run.username}
            {run.run_as && <> · as {run.run_as}</>}
            {finished && <> · {formatDuration(run.duration_ms)}</>}
            {" · "}
            {formatDistanceToNow(new Date(run.started_at * 1000), { addSuffix: true })}
          </span>
        </div>
        {run.status === "running" && (
          <Button size="sm" variant="outline" onClick={cancel} disabled={cancelling}>
            <Square className="mr-1 h-3 w-3" />
            Cancel
          </Button>
        )}
      </div>
      {run.error && <p className="text-sm text-destructive">{run.error}</p>}
      <pre
        ref={outputRef}
        onScroll={(e) => {
          const el = e.currentTarget;
          followRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
        }}
        className="max-h-96 min-h-24 overflow-auto whitespace-pre-wrap break-all border bg-muted p-3 font-mono text-xs"
      >
        {output.length === 0 && finished ? (
          <span className="text-muted-foreground">No output</span>
        ) : (
          output.map((o) => (
            <span key={o.seq} className={cn(o.stream === "stderr" && "text-destructive")}>
              {o.data}
            </span>
          ))
        )}
      </pre>
      {run.truncated && (
        <p className="text-xs text-muted-foreground">Output was cut off after 4 MiB.</p>
      )}
    </div>
  );
}
