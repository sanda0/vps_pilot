import { useCallback, useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, Play } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { CommandRunHistory } from "@/components/command-run-history";
import { CommandRunPanel } from "@/components/command-run-panel";
import { useToast } from "@/hooks/use-toast";
import { apiErrorMessage, commandRunsApi } from "@/lib/api";
import { CommandRun } from "@/types/command_run";
import { Project } from "@/types/project";

// Runs shown per page of the history.
const PAGE_SIZE = 10;

// A project's commands with their Run buttons, the selected run's output,
// and the history of runs, a page at a time.
export function ProjectCommands({ project }: { project: Project }) {
  const { toast } = useToast();
  const [runs, setRuns] = useState<CommandRun[]>([]);
  const [hasOlder, setHasOlder] = useState(false);
  const [offset, setOffset] = useState(0);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [confirmCommand, setConfirmCommand] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);

  const fetchRuns = useCallback(async () => {
    try {
      // One more than a page, to know whether there is an older page.
      const list = await commandRunsApi.list(project.id, PAGE_SIZE + 1, offset);
      setRuns(list.slice(0, PAGE_SIZE));
      setHasOlder(list.length > PAGE_SIZE);
      // Show the latest run on first load, e.g. one still running.
      setSelectedRunId((current) => current ?? list[0]?.id ?? null);
    } catch {
      // The history is secondary; the commands still work without it.
    }
  }, [project.id, offset]);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  const startRun = async () => {
    if (!confirmCommand) return;
    setStarting(true);
    try {
      const run = await commandRunsApi.start(project.id, confirmCommand);
      setSelectedRunId(run.id);
      // The new run is on the first page.
      if (offset === 0) fetchRuns();
      else setOffset(0);
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Couldn't run the command",
        description: apiErrorMessage(err, "The hub refused to start it"),
      });
    } finally {
      setStarting(false);
      setConfirmCommand(null);
    }
  };

  const commands = project.commands ?? [];
  const page = offset / PAGE_SIZE + 1;
  return (
    <div className="space-y-4">
      {commands.length === 0 && <p className="text-sm text-muted-foreground">No commands defined</p>}
      <ul className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {commands.map((cmd) => (
          <li
            key={cmd.name}
            className="flex items-center justify-between gap-3 rounded-md border bg-muted/40 p-3"
          >
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{cmd.name}</p>
              <p className="truncate font-mono text-xs text-muted-foreground">{cmd.command}</p>
            </div>
            <Button size="sm" variant="outline" onClick={() => setConfirmCommand(cmd.name)}>
              <Play className="mr-1 h-3 w-3" />
              Run
            </Button>
          </li>
        ))}
      </ul>

      {selectedRunId ? (
        <CommandRunPanel runId={selectedRunId} onFinished={fetchRuns} />
      ) : (
        commands.length > 0 && <p className="text-sm text-muted-foreground">Run a command to see its output here.</p>
      )}

      <CommandRunHistory runs={runs} selectedId={selectedRunId} onSelect={setSelectedRunId} />
      {(offset > 0 || hasOlder) && (
        <div className="flex items-center justify-end gap-2 text-sm text-muted-foreground">
          <span>Page {page}</span>
          <Button
            size="sm"
            variant="outline"
            disabled={offset === 0}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
          >
            <ChevronLeft className="mr-1 h-3 w-3" />
            Newer
          </Button>
          <Button size="sm" variant="outline" disabled={!hasOlder} onClick={() => setOffset(offset + PAGE_SIZE)}>
            Older
            <ChevronRight className="ml-1 h-3 w-3" />
          </Button>
        </div>
      )}

      <AlertDialog open={confirmCommand !== null} onOpenChange={(open) => !open && setConfirmCommand(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Run {confirmCommand}?</AlertDialogTitle>
            <AlertDialogDescription>
              Runs{" "}
              <code className="font-mono">
                {commands.find((c) => c.name === confirmCommand)?.command}
              </code>{" "}
              in {project.path} on {project.node_name || `node ${project.node_id}`}, as the owner of its
              config.vpspilot.json.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={starting}>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={startRun} disabled={starting}>
              {starting ? "Starting..." : "Run"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
