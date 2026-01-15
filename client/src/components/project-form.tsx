import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { CreateProjectSchema, createProjectSchema, UpdateProjectSchema, updateProjectSchema } from '@/schema/project';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useEffect, useState } from 'react';
import api from '@/lib/api';
import { Loader2 } from 'lucide-react';

interface Node {
  id: number;
  name: string;
  ip: string;
}

interface ProjectFormProps {
  defaultValues?: Partial<CreateProjectSchema | UpdateProjectSchema>;
  onSubmit: (data: CreateProjectSchema | UpdateProjectSchema) => Promise<void>;
  isLoading?: boolean;
  mode: 'create' | 'edit';
}

export function ProjectForm({ defaultValues, onSubmit, isLoading = false, mode }: ProjectFormProps) {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loadingNodes, setLoadingNodes] = useState(true);

  const form = useForm<CreateProjectSchema>(
    mode === 'create'
      ? {
          resolver: zodResolver(createProjectSchema),
          defaultValues: {
            name: '',
            description: '',
            node_id: undefined,
            repo_url: '',
            branch: 'main',
            deploy_path: '',
            ...defaultValues,
          },
        }
      : {
          resolver: zodResolver(updateProjectSchema) as any,
          defaultValues: {
            name: '',
            description: '',
            repo_url: '',
            branch: 'main',
            deploy_path: '',
            ...defaultValues,
          },
        }
  );

  // Load nodes on mount
  useEffect(() => {
    const fetchNodes = async () => {
      try {
        const response = await api.get('/nodes', {
          params: { limit: 100, page: 1, search: '' }
        });
        // The API returns nodes in response.data.data
        const nodesList = response.data?.data || [];
        setNodes(nodesList);
      } catch (error) {
        console.error('Failed to fetch nodes:', error);
        setNodes([]);
      } finally {
        setLoadingNodes(false);
      }
    };

    fetchNodes();
  }, []);

  const handleSubmit = async (data: CreateProjectSchema | UpdateProjectSchema) => {
    await onSubmit(data);
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Project Name *</FormLabel>
              <FormControl>
                <Input placeholder="my-awesome-project" {...field} />
              </FormControl>
              <FormDescription>
                A unique name for your project
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="description"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Description</FormLabel>
              <FormControl>
                <Textarea
                  placeholder="Brief description of your project..."
                  className="resize-none"
                  rows={3}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        {mode === 'create' && (
          <FormField
            control={form.control}
            name="node_id"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Node *</FormLabel>
                <Select
                  onValueChange={(value) => field.onChange(parseInt(value, 10))}
                  value={field.value?.toString()}
                  disabled={loadingNodes}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder={loadingNodes ? "Loading nodes..." : "Select a node"} />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {nodes.map((node) => (
                      <SelectItem key={node.id} value={node.id.toString()}>
                        {node.name || node.ip} ({node.ip})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormDescription>
                  The node where this project will be deployed
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        )}

        <FormField
          control={form.control}
          name="repo_url"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Repository URL</FormLabel>
              <FormControl>
                <Input
                  placeholder="https://github.com/username/repo.git"
                  {...field}
                />
              </FormControl>
              <FormDescription>
                Optional: Git repository URL for deployment
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="branch"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Branch *</FormLabel>
              <FormControl>
                <Input placeholder="main" {...field} />
              </FormControl>
              <FormDescription>
                Git branch to deploy
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="deploy_path"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Deploy Path *</FormLabel>
              <FormControl>
                <Input placeholder="/var/www/my-project" {...field} />
              </FormControl>
              <FormDescription>
                Absolute path on the node where the project will be deployed
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="flex gap-4 pt-4">
          <Button type="submit" disabled={isLoading}>
            {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {mode === 'create' ? 'Create Project' : 'Update Project'}
          </Button>
        </div>
      </form>
    </Form>
  );
}
