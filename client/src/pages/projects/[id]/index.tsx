import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router';
import { projectsApi } from '@/lib/api';
import { Project } from '@/types/project';
import { ProjectStatusBadge } from '@/components/project-status-badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { ArrowLeft, Pencil, Trash2, Server, FolderGit2, GitBranch, FolderTree, Calendar, Clock } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useNavigate } from 'react-router';
import { ProjectDeleteDialog } from '@/components/project-delete-dialog';
import { formatDistanceToNow, format } from 'date-fns';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';

export default function ProjectDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const { toast } = useToast();
  const navigate = useNavigate();

  useEffect(() => {
    if (id) {
      fetchProject();
    }
  }, [id]);

  const fetchProject = async () => {
    if (!id) return;
    
    try {
      setLoading(true);
      const data = await projectsApi.get(id);
      setProject(data);
    } catch (error) {
      console.error('Failed to fetch project:', error);
      toast({
        variant: 'destructive',
        title: 'Error',
        description: 'Failed to load project details',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!id) return;

    setDeleting(true);
    try {
      await projectsApi.delete(id);
      toast({
        title: 'Success',
        description: 'Project deleted successfully',
      });
      navigate('/projects');
    } catch (error) {
      console.error('Failed to delete project:', error);
      toast({
        variant: 'destructive',
        title: 'Error',
        description: 'Failed to delete project',
      });
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex-1 space-y-6 p-8">
        <div className="flex items-center justify-center py-12">
          <div className="animate-pulse">Loading...</div>
        </div>
      </div>
    );
  }

  if (!project) {
    return (
      <div className="flex-1 space-y-6 p-8">
        <div className="text-center py-12">
          <p className="text-muted-foreground">Project not found</p>
          <Button asChild className="mt-4">
            <Link to="/projects">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Projects
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 space-y-6 p-8">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink href="/">Home</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink href="/projects">Projects</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{project.name}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="icon" asChild>
              <Link to="/projects">
                <ArrowLeft className="h-4 w-4" />
              </Link>
            </Button>
            <div>
              <h1 className="text-3xl font-bold tracking-tight">{project.name}</h1>
              <p className="text-muted-foreground">{project.description || 'No description'}</p>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <ProjectStatusBadge status={project.status} />
          <Button variant="outline" asChild>
            <Link to={`/projects/${project.id}/edit`}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit
            </Link>
          </Button>
          <Button variant="destructive" onClick={() => setDeleteDialogOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Project Information</CardTitle>
            <CardDescription>Basic project configuration</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-start gap-3">
              <Server className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div className="space-y-1">
                <p className="text-sm font-medium">Node</p>
                <p className="text-sm text-muted-foreground">
                  {project.node_name || `Node ${project.node_id}`}
                  {project.node_ip && <span className="ml-2">({project.node_ip})</span>}
                </p>
              </div>
            </div>

            <Separator />

            {project.repo_url && (
              <>
                <div className="flex items-start gap-3">
                  <FolderGit2 className="h-5 w-5 mt-0.5 text-muted-foreground" />
                  <div className="space-y-1 flex-1">
                    <p className="text-sm font-medium">Repository</p>
                    <p className="text-sm text-muted-foreground break-all">{project.repo_url}</p>
                  </div>
                </div>
                <Separator />
              </>
            )}

            <div className="flex items-start gap-3">
              <GitBranch className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div className="space-y-1">
                <p className="text-sm font-medium">Branch</p>
                <p className="text-sm text-muted-foreground">{project.branch}</p>
              </div>
            </div>

            <Separator />

            <div className="flex items-start gap-3">
              <FolderTree className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div className="space-y-1">
                <p className="text-sm font-medium">Deploy Path</p>
                <p className="text-sm text-muted-foreground font-mono bg-muted px-2 py-1 rounded">
                  {project.deploy_path}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Deployment Status</CardTitle>
            <CardDescription>Current deployment information</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-start gap-3">
              <Calendar className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div className="space-y-1">
                <p className="text-sm font-medium">Created</p>
                <p className="text-sm text-muted-foreground">
                  {format(new Date(project.created_at), 'PPP')}
                </p>
              </div>
            </div>

            <Separator />

            <div className="flex items-start gap-3">
              <Clock className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div className="space-y-1">
                <p className="text-sm font-medium">Last Updated</p>
                <p className="text-sm text-muted-foreground">
                  {formatDistanceToNow(new Date(project.updated_at), { addSuffix: true })}
                </p>
              </div>
            </div>

            {project.last_deployed_at && (
              <>
                <Separator />
                <div className="flex items-start gap-3">
                  <Clock className="h-5 w-5 mt-0.5 text-muted-foreground" />
                  <div className="space-y-1">
                    <p className="text-sm font-medium">Last Deployed</p>
                    <p className="text-sm text-muted-foreground">
                      {formatDistanceToNow(new Date(project.last_deployed_at), { addSuffix: true })}
                    </p>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Deployment Logs</CardTitle>
          <CardDescription>View deployment history and logs</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            <p>Deployment logs will be available once the GitHub integration is set up</p>
          </div>
        </CardContent>
      </Card>

      <ProjectDeleteDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onConfirm={handleDelete}
        projectName={project.name}
        isLoading={deleting}
      />
    </div>
  );
}
