export interface ProjectCommand {
  name: string;
  command: string;
}

export interface ProjectBackupDatabase {
  connection: string;
  host: string;
  port: string;
  username: string;
  password: string;
  database_name: string;
}

// Where backups are uploaded. The keys are the names of env-file variables,
// never the keys themselves.
export interface ProjectBackupS3 {
  access_key: string;
  secret_key: string;
  bucket: string;
  region: string;
  endpoint: string;
  prefix?: string;
  path_style: boolean;
  retention_days?: number; // default 30
}

export interface ProjectBackup {
  env_file?: string;
  zip_file_name?: string;
  database?: ProjectBackupDatabase;
  dir?: string[];
  schedule?: "daily" | "weekly"; // weekly runs on Sundays
  at?: string; // HH:MM in the server's time zone, default 03:00
  local_dir?: string;
  retention_days?: number; // default 7
  s3?: ProjectBackupS3;
}

export interface Project {
  id: string;
  node_id: number;
  node_name?: string;
  node_ip?: string;
  name: string;
  path: string;
  tech: string[];
  commands: ProjectCommand[];
  logs: string[];
  backups?: ProjectBackup;
  discovered_at: string;
  updated_at: string;
}

export interface ProjectListResponse {
  data: Project[];
  total: number;
  limit: number;
  offset: number;
}
