// A node's project search settings and its last scan
// (GET /nodes/:id/project-settings).

export interface ProjectScanError {
  path: string;
  error: string;
}

export interface ProjectPath {
  id: number;
  path: string;
  created_at: number; // unix seconds
}

export interface ProjectScan {
  node_id: number;
  auto_scan: boolean;
  scan_interval: number; // seconds, 60..86400
  last_scanned_at: number | null; // unix seconds; null until the agent first reports
  errors: ProjectScanError[];
  paths: ProjectPath[];
  online: boolean;
}

export interface NodeOption {
  id: number;
  name: string;
  ip: string;
}
