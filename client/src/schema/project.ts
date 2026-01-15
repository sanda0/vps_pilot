import { z } from 'zod';

export const createProjectSchema = z.object({
  name: z.string().min(1, 'Project name is required').max(100, 'Name must be less than 100 characters'),
  description: z.string().max(500, 'Description must be less than 500 characters').optional(),
  node_id: z.number().int().positive('Please select a node'),
  repo_url: z.string().url('Must be a valid URL').optional().or(z.literal('')),
  branch: z.string().min(1, 'Branch is required').default('main'),
  deploy_path: z.string().min(1, 'Deploy path is required'),
});

export const updateProjectSchema = z.object({
  name: z.string().min(1, 'Project name is required').max(100, 'Name must be less than 100 characters'),
  description: z.string().max(500, 'Description must be less than 500 characters').optional(),
  repo_url: z.string().url('Must be a valid URL').optional().or(z.literal('')),
  branch: z.string().min(1, 'Branch is required'),
  deploy_path: z.string().min(1, 'Deploy path is required'),
  status: z.enum(['inactive', 'cloning', 'active', 'error']).optional(),
});

export type CreateProjectSchema = z.infer<typeof createProjectSchema>;
export type UpdateProjectSchema = z.infer<typeof updateProjectSchema>;
