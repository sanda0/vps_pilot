import { useCallback, useEffect, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { apiErrorMessage, cronApi } from "@/lib/api";
import { describeSchedule } from "@/lib/cron";
import { cn, formatDuration } from "@/lib/utils";
import { CronJob, CronRun } from "@/types/cron";
import { Project } from "@/types/project";

// Jobs can run every minute, so the list refreshes about twice as often.
const POLL_MS = 30_000;

function CronStatusBadge({ run }: { run: CronRun }) {
  if (run.status === "exited") {
    return run.exit_code === 0 ? (
      <Badge variant="outline">Exit 0</Badge>
    ) : (
      <Badge variant="destructive">Exit {run.exit_code}</Badge>
    );
  }
  if (run.status === "skipped") return <Badge variant="secondary">Skipped</Badge>;
  if (run.status === "timeout") return <Badge variant="destructive">Timed out</Badge>;
  const label = run.status.charAt(0).toUpperCase() + run.status.slice(1);
  return <Badge variant="destructive">{label}</Badge>;
}

function ago(unix: number): string {
  return formatDistanceToNow(new Date(unix * 1000), { addSuffix: true });
}

function RunLine({ run }: { run: CronRun }) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <CronStatusBadge run={run} />
      <span className="text-xs text-muted-foreground">
        {ago(run.scheduled_at)}
        {run.duration_ms > 0 && <> · {formatDuration(run.duration_ms)}</>}
        {run.run_as && <> · as {run.run_as}</>}
      </span>
    </div>
  );
}

function RunOutput({ run }: { run: CronRun }) {
  const [open, setOpen] = useState(false);
  if (!run.output) return null;
  return (
    <div className="space-y-1">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
      >
        {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        Output{run.truncated && " (last 64 KiB)"}
      </button>
      {open && (
        <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-all border bg-muted p-3 font-mono text-xs">
          {run.output}
        </pre>
      )}
    </div>
  );
}

function RunDetails({ run }: { run: CronRun }) {
  return (
    <div className="space-y-1">
      <RunLine run={run} />
      {run.error && (
        // A skipped run is expected when a job overlaps itself, not a failure.
        <p className={cn("break-words text-xs", run.status === "skipped" ? "text-muted-foreground" : "text-destructive")}>
          {run.error}
        </p>
      )}
      <RunOutput run={run} />
    </div>
  );
}

function JobHistory({ projectId, command }: { projectId: string; command: string }) {
  const [runs, setRuns] = useState<CronRun[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    cronApi
      .runs(projectId, command)
      .then(setRuns)
      .catch((err) => setError(apiErrorMessage(err, "Couldn't load the runs")));
  }, [projectId, command]);

  if (error) return <p className="text-sm text-destructive">{error}</p>;
  if (runs === null) return <p className="text-sm text-muted-foreground">Loading…</p>;
  return (
    <ul className="divide-y border">
      {runs.map((run) => (
        <li key={run.id} className="px-3 py-2">
          <RunDetails run={run} />
        </li>
      ))}
    </ul>
  );
}

function JobRow({ project, job }: { project: Project; job: CronJob }) {
  const [history, setHistory] = useState(false);
  const words = describeSchedule(job.schedule);
  const command = project.commands.find((c) => c.name === job.command)?.command;
  return (
    <li className="space-y-2 px-3 py-3 text-sm">
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <span className="font-medium">{job.command}</span>
        <code className="font-mono text-xs">{job.schedule}</code>
        {words && <span className="text-xs text-muted-foreground">{words}, server time</span>}
        {job.last_run && (
          <Button variant="ghost" size="sm" className="ml-auto h-7" onClick={() => setHistory(!history)}>
            {history ? "Hide history" : "History"}
          </Button>
        )}
      </div>
      {command && <p className="break-all font-mono text-xs text-muted-foreground">{command}</p>}
      {job.last_run ? (
        <RunDetails run={job.last_run} />
      ) : (
        <p className="text-xs text-muted-foreground">Hasn't run yet</p>
      )}
      {history && <JobHistory projectId={project.id} command={job.command} />}
    </li>
  );
}

// A project's cron jobs, each with its last run, and their history on
// demand. The agent runs them; Run in Commands runs one now.
export function ProjectCron({ project }: { project: Project }) {
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [error, setError] = useState("");

  const fetchJobs = useCallback(async () => {
    try {
      setJobs(await cronApi.jobs(project.id));
      setError("");
    } catch (err) {
      setError(apiErrorMessage(err, "Couldn't load the cron jobs"));
    }
  }, [project.id]);

  useEffect(() => {
    fetchJobs();
    const timer = setInterval(fetchJobs, POLL_MS);
    return () => clearInterval(timer);
  }, [fetchJobs]);

  return (
    <div className="space-y-3">
      {error && <p className="text-sm text-destructive">{error}</p>}
      <ul className="divide-y border">
        {jobs.map((job) => (
          <JobRow key={job.command} project={project} job={job} />
        ))}
      </ul>
    </div>
  );
}
