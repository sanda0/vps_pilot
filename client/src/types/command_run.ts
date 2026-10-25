// A command run from the hub's audit log (GET /runs/:id).
export type CommandRunStatus =
  | "running"
  | "exited"
  | "timeout"
  | "cancelled"
  | "rejected"
  | "failed"
  | "lost";

export interface CommandRun {
  id: string;
  node_id: number;
  project_path: string;
  project_name: string;
  command: string;
  command_line: string;
  username: string;
  status: CommandRunStatus;
  exit_code: number | null;
  run_as: string;
  error: string;
  truncated: boolean;
  duration_ms: number;
  started_at: number; // Unix seconds
  finished_at: number | null;
}

export interface CommandOutput {
  seq: number;
  stream: "stdout" | "stderr";
  data: string;
}

// One message on /runs/:id/ws.
export type CommandRunMessage = { run: CommandRun } | { output: CommandOutput };
