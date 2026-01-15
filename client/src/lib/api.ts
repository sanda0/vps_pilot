
import axios from 'axios';
import { Project, CreateProjectInput, UpdateProjectInput, ProjectListResponse } from '@/types/project';


const api = axios.create({
  baseURL: "http://localhost:8000/api/v1",
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
});

// Projects API
export const projectsApi = {
  list: async (limit = 10, offset = 0) => {
    const response = await api.get<ProjectListResponse>('/projects', {
      params: { limit, offset }
    });
    return response.data;
  },

  get: async (id: string) => {
    const response = await api.get<Project>(`/projects/${id}`);
    return response.data;
  },

  create: async (data: CreateProjectInput) => {
    const response = await api.post<Project>('/projects', data);
    return response.data;
  },

  update: async (id: string, data: UpdateProjectInput) => {
    const response = await api.put<Project>(`/projects/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    const response = await api.delete(`/projects/${id}`);
    return response.data;
  },

  listByNode: async (nodeId: number, limit = 10, offset = 0) => {
    const response = await api.get<ProjectListResponse>(`/nodes/${nodeId}/projects`, {
      params: { limit, offset }
    });
    return response.data;
  },
};

// GitHub API
export const githubApi = {
  saveToken: async (token: string) => {
    const response = await api.post('/github/token', { token });
    return response.data;
  },

  getRepos: async () => {
    const response = await api.get('/github/repos');
    return response.data;
  },

  getStatus: async () => {
    const response = await api.get('/github/status');
    return response.data;
  },

  deleteToken: async () => {
    const response = await api.delete('/github/token');
    return response.data;
  },
};

export default api;