// Cron jobs a project declares and their runs (GET /projects/:id/cron).
// The node's agent runs them; the dashboard only shows what it reported.
export type CronRunStatus = "exited" | "timeout" | "skipped" | "rejected" | "failed";

export interface CronRun {
  id: string;
  command: string;
  schedule: string;
  scheduled_at: number; // the minute it was due, Unix seconds
  status: CronRunStatus;
  exit_code: number | null; // null unless the command ran
  run_as: string;
  duration_ms: number;
  output: string; // last 64 KiB of stdout and stderr together
  truncated: boolean;
  error: string;
  received_at: number; // Unix seconds
}

export interface CronJob {
  command: string;
  schedule: string;
  last_run: CronRun | null;
}
