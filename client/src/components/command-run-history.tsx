import { formatDistanceToNow } from "date-fns";
import { RunStatusBadge } from "@/components/command-run-panel";
import { cn, formatDuration } from "@/lib/utils";
import { CommandRun } from "@/types/command_run";

interface CommandRunHistoryProps {
  runs: CommandRun[];
  selectedId: string | null;
  onSelect: (runId: string) => void;
}

// A project's recent runs, newest first. Selecting one shows its output.
export function CommandRunHistory({ runs, selectedId, onSelect }: CommandRunHistoryProps) {
  if (runs.length === 0) {
    return <p className="py-4 text-center text-sm text-muted-foreground">No runs yet</p>;
  }
  return (
    <ul className="divide-y border">
      {runs.map((run) => (
        <li key={run.id}>
          <button
            type="button"
            onClick={() => onSelect(run.id)}
            className={cn(
              "flex w-full flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-left text-sm hover:bg-muted/60",
              run.id === selectedId && "bg-muted",
            )}
          >
            <span className="font-medium">{run.command}</span>
            <RunStatusBadge run={run} />
            <span className="ml-auto text-xs text-muted-foreground">
              {run.username}
              {run.status !== "running" && <> · {formatDuration(run.duration_ms)}</>}
              {" · "}
              {formatDistanceToNow(new Date(run.started_at * 1000), { addSuffix: true })}
            </span>
          </button>
        </li>
      ))}
    </ul>
  );
}
