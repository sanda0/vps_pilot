import axios from "axios";
import { Project, ProjectListResponse } from "@/types/project";
import { JoinToken } from "@/types/node_type";

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

// Projects API — read-only from the dashboard perspective.
// Projects are created/updated exclusively by agents via /api/v1/agent/projects/sync
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
};

export default api;
