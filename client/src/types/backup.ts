// A backup of a project (GET /projects/:id/backups), scheduled by the
// agent or started with Back up now.
export type BackupStatus = "running" | "ok" | "failed" | "rejected" | "lost";

export interface BackupRun {
  id: string;
  node_id: number;
  project_path: string;
  trigger: "manual" | "schedule";
  username: string; // who pressed Back up now
  status: BackupStatus;
  run_as: string;
  file: string;
  local_path: string;
  size: number; // bytes
  database: string; // mysql, postgres, sqlite or empty
  upload_status: "ok" | "failed" | "skipped" | "";
  upload_key: string;
  upload_error: string;
  error: string;
  duration_ms: number;
  started_at: number; // Unix seconds
  finished_at: number | null;
}
