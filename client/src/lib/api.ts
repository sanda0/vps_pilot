import axios from "axios";
import { Project, ProjectListResponse } from "@/types/project";
import { JoinToken } from "@/types/node_type";
import { NodeOption, ProjectPath, ProjectScan } from "@/types/project_scan";

const api = axios.create({
  baseURL: "/api/v1",
  headers: {
    "Content-Type": "application/json",
  },
  withCredentials: true,
});

// Attach JWT token to every request if present
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Projects API. Projects themselves come from agents (`projects` messages);
// the dashboard can only point an agent at extra directories (projectScanApi).
export const projectsApi = {
  list: async (limit = 10, offset = 0) => {
    const response = await api.get<ProjectListResponse>("/projects", {
      params: { limit, offset },
    });
    return response.data;
  },

  get: async (id: string) => {
    const response = await api.get<Project>(`/projects/${id}`);
    return response.data;
  },

  delete: async (id: string) => {
    const response = await api.delete(`/projects/${id}`);
    return response.data;
  },

  listByNode: async (nodeId: number, limit = 10, offset = 0) => {
    const response = await api.get<ProjectListResponse>(
      `/nodes/${nodeId}/projects`,
      {
        params: { limit, offset },
      },
    );
    return response.data;
  },
};

// Agent enrollment
export const nodesApi = {
  // nodeId set = re-enroll that node instead of adding a new one
  createJoinToken: async (nodeId?: number) => {
    const response = await api.post<{ data: JoinToken }>(
      "/nodes/join-tokens",
      nodeId ? { node_id: nodeId } : {},
    );
    return response.data.data;
  },

  revokeKey: async (nodeId: number) => {
    const response = await api.delete(`/nodes/${nodeId}/key`);
    return response.data;
  },

  list: async () => {
    const response = await api.get<{ data: NodeOption[] }>("/nodes", {
      params: { limit: 200, page: 1, search: "" },
    });
    return response.data.data ?? [];
  },
};

// How each node searches for projects, plus project directories added by hand.
export const projectScanApi = {
  get: async (nodeId: number) => {
    const response = await api.get<{ data: ProjectScan }>(`/nodes/${nodeId}/project-settings`);
    return response.data.data;
  },

  update: async (nodeId: number, autoScan: boolean, scanInterval: number) => {
    const response = await api.put<{ data: ProjectScan }>(`/nodes/${nodeId}/project-settings`, {
      auto_scan: autoScan,
      scan_interval: scanInterval,
    });
    return response.data.data;
  },

  rescan: async (nodeId: number) => {
    await api.post(`/nodes/${nodeId}/rescan-projects`);
  },

  addPath: async (nodeId: number, path: string) => {
    const response = await api.post<{ data: ProjectPath }>(`/nodes/${nodeId}/project-paths`, { path });
    return response.data.data;
  },

  deletePath: async (nodeId: number, pathId: number) => {
    await api.delete(`/nodes/${nodeId}/project-paths/${pathId}`);
  },
};

// The hub's error message from a failed request, or `fallback`.
export function apiErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err) && typeof err.response?.data?.error === "string") {
    return err.response.data.error;
  }
  return fallback;
}

export default api;
