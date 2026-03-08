import axios from "axios";
import { Project, ProjectListResponse } from "@/types/project";

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

// GitHub API
export const githubApi = {
  saveToken: async (token: string) => {
    const response = await api.post("/github/token", { token });
    return response.data;
  },

  getRepos: async () => {
    const response = await api.get("/github/repos");
    return response.data;
  },

  getStatus: async () => {
    const response = await api.get("/github/status");
    return response.data;
  },

  deleteToken: async () => {
    const response = await api.delete("/github/token");
    return response.data;
  },
};

export default api;
