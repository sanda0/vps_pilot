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
import api, { githubApi } from '@/lib/api';
import { Github, Loader2, ExternalLink } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { useNavigate } from 'react-router';

interface Node {
  id: number;
  name: string;
  ip: string;
}

interface GitHubRepo {
  id: number;
  name: string;
  full_name: string;
  private: boolean;
  html_url: string;
  clone_url: string;
  ssh_url: string;
  description: string;
  default_branch: string;
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
  const [githubRepos, setGithubRepos] = useState<GitHubRepo[]>([]);
  const [loadingGithubRepos, setLoadingGithubRepos] = useState(false);
  const [githubConnected, setGithubConnected] = useState(false);
  const navigate = useNavigate();

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

  // Check GitHub connection and load repos
  useEffect(() => {
    const checkGitHubConnection = async () => {
      try {
        setLoadingGithubRepos(true);
        const status = await githubApi.getStatus();
        setGithubConnected(status.connected);
        
        if (status.connected) {
          const reposResponse = await githubApi.getRepos();
          setGithubRepos(reposResponse.data || []);
        }
      } catch (error) {
        console.error('Failed to check GitHub connection:', error);
        setGithubConnected(false);
      } finally {
        setLoadingGithubRepos(false);
      }
    };

    checkGitHubConnection();
  }, []);

  const handleSubmit = async (data: CreateProjectSchema | UpdateProjectSchema) => {
    await onSubmit(data);
  };

  const handleSelectGitHubRepo = (repoCloneUrl: string, defaultBranch: string) => {
    form.setValue('repo_url', repoCloneUrl);
    form.setValue('branch', defaultBranch || 'main');
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

        {/* GitHub Repository Selection */}
        {mode === 'create' && (
          <div className="space-y-4">
            {!githubConnected && !loadingGithubRepos && (
              <Alert>
                <Github className="h-4 w-4" />
                <AlertTitle>Connect GitHub (Optional)</AlertTitle>
                <AlertDescription className="flex flex-col gap-2">
                  <p>Connect your GitHub account to easily select repositories</p>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    type="button"
                    onClick={() => navigate('/settings/github')}
                  >
                    <Github className="mr-2 h-4 w-4" />
                    Connect GitHub
                    <ExternalLink className="ml-2 h-3 w-3" />
                  </Button>
                </AlertDescription>
              </Alert>
            )}

            {loadingGithubRepos && (
              <div className="flex items-center justify-center p-4 border rounded-lg">
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                <span className="text-sm text-muted-foreground">Loading GitHub repositories...</span>
              </div>
            )}

            {githubConnected && !loadingGithubRepos && (
              <FormField
                control={form.control}
                name="repo_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>GitHub Repository</FormLabel>
                    <Select
                      onValueChange={(value) => {
                        const repo = githubRepos.find(r => r.clone_url === value);
                        if (repo) {
                          handleSelectGitHubRepo(repo.clone_url, repo.default_branch);
                        }
                      }}
                      value={field.value}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select a repository" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {githubRepos.map((repo) => (
                          <SelectItem key={repo.id} value={repo.clone_url}>
                            <div className="flex items-center gap-2">
                              <Github className="h-4 w-4" />
                              <span>{repo.full_name}</span>
                              {repo.private && (
                                <span className="text-xs text-muted-foreground">(Private)</span>
                              )}
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      Or enter a repository URL manually below
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </div>
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
