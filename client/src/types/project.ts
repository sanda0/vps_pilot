export type ProjectStatus = 'inactive' | 'cloning' | 'active' | 'error';

export interface Project {
  id: string;
  name: string;
  description: string;
  node_id: number;
  node_name?: string;
  node_ip?: string;
  repo_url: string;
  branch: string;
  deploy_path: string;
  status: ProjectStatus;
  last_deployed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectInput {
  name: string;
  description?: string;
  node_id: number;
  repo_url?: string;
  branch: string;
  deploy_path: string;
}

export interface UpdateProjectInput {
  name: string;
  description?: string;
  repo_url?: string;
  branch: string;
  deploy_path: string;
  status?: ProjectStatus;
}

export interface ProjectListResponse {
  data: Project[];
  total: number;
  limit: number;
  offset: number;
}
