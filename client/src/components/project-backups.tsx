import { useCallback, useEffect, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { AlertTriangle, Archive, Loader2 } from "lucide-react";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { apiErrorMessage, backupsApi } from "@/lib/api";
import { formatBytes, formatDuration } from "@/lib/utils";
import { BackupRun } from "@/types/backup";
import { Project, ProjectBackup } from "@/types/project";

// How often the list refreshes while a backup is running.
const POLL_MS = 3000;

function scheduleText(b: ProjectBackup): string {
  const at = b.at || "03:00";
  if (b.schedule === "daily") return `Daily at ${at}, server time`;
  if (b.schedule === "weekly") return `Sundays at ${at}, server time`;
  return "None: backups run only from Back up now";
}

function includesText(b: ProjectBackup): string {
  const parts = [...(b.dir ?? [])];
  if (b.env_file) parts.push(b.env_file);
  if (b.database) parts.push(`the database (${b.database.connection})`);
  return parts.length > 0 ? parts.join(", ") : "Nothing declared";
}

function BackupStatusBadge({ run }: { run: BackupRun }) {
  if (run.status === "running") {
    return (
      <Badge variant="secondary" className="gap-1">
        <Loader2 className="h-3 w-3 animate-spin" />
        Running
      </Badge>
    );
  }
  if (run.status === "ok") return <Badge variant="outline">OK</Badge>;
  const label = run.status.charAt(0).toUpperCase() + run.status.slice(1);
  return <Badge variant="destructive">{label}</Badge>;
}

function BackupHistory({ runs }: { runs: BackupRun[] }) {
  if (runs.length === 0) {
    return <p className="py-4 text-center text-sm text-muted-foreground">No backups yet</p>;
  }
  return (
    <ul className="divide-y border">
      {runs.map((run) => (
        <li key={run.id} className="space-y-1 px-3 py-2 text-sm">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <BackupStatusBadge run={run} />
            <span className="font-medium">
              {run.trigger === "schedule" ? "Scheduled" : `By ${run.username || "a deleted user"}`}
            </span>
            <span className="ml-auto text-xs text-muted-foreground">
              {run.size > 0 && <>{formatBytes(run.size)} · </>}
              {run.duration_ms > 0 && <>{formatDuration(run.duration_ms)} · </>}
              {formatDistanceToNow(new Date(run.started_at * 1000), { addSuffix: true })}
            </span>
          </div>
          {run.local_path && <p className="break-all font-mono text-xs text-muted-foreground">{run.local_path}</p>}
          {run.upload_status === "ok" && (
            <p className="break-words text-xs text-muted-foreground">Uploaded as {run.upload_key}</p>
          )}
          {run.upload_status === "failed" && (
            <p className="break-words text-xs text-destructive">Upload failed: {run.upload_error}</p>
          )}
          {run.error && <p className="break-words text-xs text-destructive">{run.error}</p>}
        </li>
      ))}
    </ul>
  );
}

// What a project backs up and where, Back up now, and its recent backups.
export function ProjectBackups({ project, backups }: { project: Project; backups: ProjectBackup }) {
  const { toast } = useToast();
  const [runs, setRuns] = useState<BackupRun[]>([]);
  const [error, setError] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [starting, setStarting] = useState(false);

  const fetchRuns = useCallback(async () => {
    try {
      setRuns(await backupsApi.list(project.id));
      setError("");
    } catch (err) {
      setError(apiErrorMessage(err, "Couldn't load the backups"));
    }
  }, [project.id]);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  const running = runs.some((r) => r.status === "running");
  useEffect(() => {
    if (!running) return;
    const timer = setInterval(fetchRuns, POLL_MS);
    return () => clearInterval(timer);
  }, [running, fetchRuns]);

  const start = async () => {
    setStarting(true);
    try {
      await backupsApi.start(project.id);
      await fetchRuns();
    } catch (err) {
      toast({
        title: "Couldn't start the backup",
        description: apiErrorMessage(err, "Please try again"),
        variant: "destructive",
      });
    } finally {
      setStarting(false);
      setConfirming(false);
    }
  };

  const rows: [string, string][] = [
    ["Includes", includesText(backups)],
    ["Schedule", scheduleText(backups)],
    ["Kept on the server", backups.local_dir ? `${backups.local_dir}, for ${backups.retention_days ?? 7} days` : "—"],
    [
      "Uploaded to",
      backups.s3
        ? `s3://${backups.s3.bucket}/${backups.s3.prefix ?? ""} at ${backups.s3.endpoint}, for ${backups.s3.retention_days ?? 30} days`
        : "Not uploaded",
    ],
  ];

  return (
    <div className="space-y-4">
      <dl className="grid gap-x-4 gap-y-2 text-sm sm:grid-cols-[max-content_1fr]">
        {rows.map(([label, value]) => (
          <div key={label} className="contents">
            <dt className="text-muted-foreground">{label}</dt>
            <dd className="break-words">{value}</dd>
          </div>
        ))}
      </dl>
      {!backups.local_dir && (
        <p className="flex items-start gap-2 text-sm text-destructive">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          backups.local_dir isn't set in config.vpspilot.json, so backups fail until it is.
        </p>
      )}
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={() => setConfirming(true)} disabled={starting || running}>
          <Archive className="mr-2 h-4 w-4" />
          Back up now
        </Button>
        {running && <span className="text-sm text-muted-foreground">A backup is running…</span>}
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
      <BackupHistory runs={runs} />

      <AlertDialog open={confirming} onOpenChange={(open) => !open && !starting && setConfirming(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Back up {project.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              The agent on {project.node_name || `node ${project.node_id}`} zips what config.vpspilot.json
              declares into {backups.local_dir || "local_dir"}
              {backups.s3 && " and uploads it"}, as the owner of the config file.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={starting}>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={start} disabled={starting}>
              {starting ? "Starting..." : "Back up"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
