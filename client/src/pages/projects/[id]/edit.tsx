import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router';
import { projectsApi } from '@/lib/api';
import { Project } from '@/types/project';
import { ProjectForm } from '@/components/project-form';
import { UpdateProjectSchema } from '@/schema/project';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ArrowLeft } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';

export default function EditProjectPage() {
  const { id } = useParams<{ id: string }>();
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const navigate = useNavigate();
  const { toast } = useToast();

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

  const handleSubmit = async (data: UpdateProjectSchema) => {
    if (!id) return;

    setIsSubmitting(true);
    try {
      const updatedProject = await projectsApi.update(id, {
        name: data.name,
        description: data.description,
        repo_url: data.repo_url || '',
        branch: data.branch,
        deploy_path: data.deploy_path,
        status: data.status,
      });

      toast({
        title: 'Success',
        description: `Project "${updatedProject.name}" updated successfully`,
      });

      navigate(`/projects/${id}`);
    } catch (error: any) {
      console.error('Failed to update project:', error);
      toast({
        variant: 'destructive',
        title: 'Error',
        description: error.response?.data?.details || 'Failed to update project',
      });
    } finally {
      setIsSubmitting(false);
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
            <BreadcrumbLink href={`/projects/${id}`}>{project.name}</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Edit</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link to={`/projects/${id}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Edit Project</h1>
          <p className="text-muted-foreground">
            Update project configuration
          </p>
        </div>
      </div>

      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle>Project Details</CardTitle>
          <CardDescription>
            Modify the project information below
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ProjectForm
            mode="edit"
            defaultValues={{
              name: project.name,
              description: project.description,
              repo_url: project.repo_url,
              branch: project.branch,
              deploy_path: project.deploy_path,
              status: project.status,
            }}
            onSubmit={handleSubmit}
            isLoading={isSubmitting}
          />
        </CardContent>
      </Card>
    </div>
  );
}
