import { useState, useEffect } from "react";
import { projectsApi } from "@/lib/api";
import { Project } from "@/types/project";
import { ProjectCard } from "@/components/project-card";
import { ProjectDeleteDialog } from "@/components/project-delete-dialog";
import { Input } from "@/components/ui/input";
import { Search, Loader2, PackageSearch } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { Button } from "@/components/ui/button";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

export default function ProjectsListPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
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
      console.error("Failed to fetch projects:", error);
      toast({
        variant: "destructive",
        title: "Error",
        description: "Failed to load projects",
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleDelete = (id: string) => {
    const project = projects.find((p) => p.id === id);
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
        title: "Success",
        description: `Project "${projectToDelete.name}" removed successfully`,
      });
      setDeleteDialogOpen(false);
      setProjectToDelete(null);
      fetchProjects(currentPage * limit);
    } catch (error) {
      console.error("Failed to delete project:", error);
      toast({
        variant: "destructive",
        title: "Error",
        description: "Failed to remove project",
      });
    } finally {
      setDeleting(false);
    }
  };

  const filteredProjects = projects.filter(
    (project) =>
      project.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      project.path?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      project.node_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      project.tech?.some((t) =>
        t.toLowerCase().includes(searchTerm.toLowerCase()),
      ),
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

      <div>
        <h1 className="text-3xl font-bold tracking-tight">Projects</h1>
        <p className="text-muted-foreground">
          Projects are discovered automatically by agents scanning nodes for{" "}
          <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
            config.vpspilot.json
          </code>{" "}
          files.
        </p>
      </div>

      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search by name, path, node or tech…"
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
        <div className="flex flex-col items-center justify-center py-16 text-center gap-4">
          <PackageSearch className="h-12 w-12 text-muted-foreground" />
          <div>
            <p className="text-lg font-medium">
              {searchTerm
                ? "No projects match your search"
                : "No projects discovered yet"}
            </p>
            <p className="text-sm text-muted-foreground mt-1">
              {searchTerm
                ? "Try a different search term"
                : "Install the agent on a node and add a config.vpspilot.json file to your projects"}
            </p>
          </div>
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
                Showing {currentPage * limit + 1} to{" "}
                {Math.min((currentPage + 1) * limit, total)} of {total} projects
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
        projectName={projectToDelete?.name || ""}
        isLoading={deleting}
      />
    </div>
  );
}
