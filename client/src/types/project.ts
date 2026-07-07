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

export interface ProjectBackup {
  env_file?: string;
  zip_file_name?: string;
  database?: ProjectBackupDatabase;
  dir?: string[];
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
