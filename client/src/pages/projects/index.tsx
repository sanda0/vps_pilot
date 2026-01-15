import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { projectsApi } from '@/lib/api';
import { Project } from '@/types/project';
import { ProjectCard } from '@/components/project-card';
import { ProjectDeleteDialog } from '@/components/project-delete-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Plus, Search, Loader2 } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';

export default function ProjectsListPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [projectToDelete, setProjectToDelete] = useState<Project | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(0);
  const limit = 12;
  const { toast } = useToast();

  const fetchProjects = async (offset = 0) => {
    try {
      setLoading(true);
      const response = await projectsApi.list(limit, offset);
      setProjects(response.data);
      setTotal(response.total);
      setCurrentPage(offset / limit);
    } catch (error) {
      console.error('Failed to fetch projects:', error);
      toast({
        variant: 'destructive',
        title: 'Error',
        description: 'Failed to load projects',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleDelete = (id: string) => {
    const project = projects.find(p => p.id === id);
    if (project) {
      setProjectToDelete(project);
      setDeleteDialogOpen(true);
    }
  };

  const confirmDelete = async () => {
    if (!projectToDelete) return;

    setDeleting(true);
    try {
      await projectsApi.delete(projectToDelete.id);
      toast({
        title: 'Success',
        description: `Project "${projectToDelete.name}" deleted successfully`,
      });
      setDeleteDialogOpen(false);
      setProjectToDelete(null);
      fetchProjects(currentPage * limit);
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

  const filteredProjects = projects.filter((project) =>
    project.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    project.description?.toLowerCase().includes(searchTerm.toLowerCase()) ||
    project.node_name?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleNextPage = () => {
    const nextOffset = (currentPage + 1) * limit;
    if (nextOffset < total) {
      fetchProjects(nextOffset);
    }
  };

  const handlePrevPage = () => {
    const prevOffset = Math.max(0, (currentPage - 1) * limit);
    fetchProjects(prevOffset);
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
            <BreadcrumbPage>Projects</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Projects</h1>
          <p className="text-muted-foreground">
            Manage your deployment projects
          </p>
        </div>
        <Button asChild>
          <Link to="/projects/create">
            <Plus className="mr-2 h-4 w-4" />
            Create Project
          </Link>
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search projects..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : filteredProjects.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-muted-foreground">
            {searchTerm ? 'No projects found matching your search' : 'No projects yet'}
          </p>
          {!searchTerm && (
            <Button asChild className="mt-4">
              <Link to="/projects/create">
                <Plus className="mr-2 h-4 w-4" />
                Create your first project
              </Link>
            </Button>
          )}
        </div>
      ) : (
        <>
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {filteredProjects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onDelete={handleDelete}
              />
            ))}
          </div>

          {total > limit && (
            <div className="flex items-center justify-between border-t pt-4">
              <div className="text-sm text-muted-foreground">
                Showing {currentPage * limit + 1} to {Math.min((currentPage + 1) * limit, total)} of {total} projects
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePrevPage}
                  disabled={currentPage === 0}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={(currentPage + 1) * limit >= total}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      <ProjectDeleteDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onConfirm={confirmDelete}
        projectName={projectToDelete?.name || ''}
        isLoading={deleting}
      />
    </div>
  );
}
