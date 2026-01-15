import { useState } from 'react';
import { useNavigate } from 'react-router';
import { projectsApi } from '@/lib/api';
import { ProjectForm } from '@/components/project-form';
import { CreateProjectSchema, UpdateProjectSchema } from '@/schema/project';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useToast } from '@/hooks/use-toast';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { Button } from '@/components/ui/button';
import { ArrowLeft } from 'lucide-react';
import { Link } from 'react-router';

export default function CreateProjectPage() {
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();
  const { toast } = useToast();

  const handleSubmit = async (data: CreateProjectSchema | UpdateProjectSchema) => {
    setIsLoading(true);
    try {
      if (!('node_id' in data)) {
        throw new Error('node_id is required for creating a project');
      }
      const project = await projectsApi.create({
        name: data.name,
        description: data.description,
        node_id: data.node_id,
        repo_url: data.repo_url || '',
        branch: data.branch,
        deploy_path: data.deploy_path,
      });

      toast({
        title: 'Success',
        description: `Project "${project.name}" created successfully`,
      });

      navigate(`/projects/${project.id}`);
    } catch (error: any) {
      console.error('Failed to create project:', error);
      toast({
        variant: 'destructive',
        title: 'Error',
        description: error.response?.data?.details || 'Failed to create project',
      });
    } finally {
      setIsLoading(false);
    }
  };

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
            <BreadcrumbPage>Create</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/projects">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Create Project</h1>
          <p className="text-muted-foreground">
            Set up a new deployment project
          </p>
        </div>
      </div>

      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle>Project Details</CardTitle>
          <CardDescription>
            Fill in the information below to create a new project
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ProjectForm
            mode="create"
            onSubmit={handleSubmit}
            isLoading={isLoading}
          />
        </CardContent>
      </Card>
    </div>
  );
}
