import { useState, useEffect } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { projectsApi } from "@/lib/api";
import { Project } from "@/types/project";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  ArrowLeft,
  Trash2,
  Server,
  FolderOpen,
  Terminal,
  ScrollText,
  Database,
  Archive,
  Clock,
  Loader2,
} from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { ProjectDeleteDialog } from "@/components/project-delete-dialog";
import { formatDistanceToNow, format } from "date-fns";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

export default function ProjectDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const { toast } = useToast();
  const navigate = useNavigate();

  useEffect(() => {
    if (id) fetchProject();
  }, [id]);

  const fetchProject = async () => {
    if (!id) return;
    try {
      setLoading(true);
      const data = await projectsApi.get(id);
      setProject(data);
    } catch {
      toast({
        variant: "destructive",
        title: "Error",
        description: "Failed to load project details",
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
      toast({ title: "Success", description: "Project removed successfully" });
      navigate("/projects");
    } catch {
      toast({
        variant: "destructive",
        title: "Error",
        description: "Failed to remove project",
      });
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
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
      {/* Breadcrumb */}
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

      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild>
            <Link to="/projects">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">
              {project.name}
            </h1>
            {project.tech && project.tech.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-2">
                {project.tech.map((t) => (
                  <Badge key={t} variant="secondary">
                    {t}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        </div>
        <Button variant="destructive" onClick={() => setDeleteDialogOpen(true)}>
          <Trash2 className="mr-2 h-4 w-4" />
          Remove
        </Button>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Project Info */}
        <Card>
          <CardHeader>
            <CardTitle>Project Info</CardTitle>
            <CardDescription>
              Discovered from config.vpspilot.json
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Node */}
            <div className="flex items-start gap-3">
              <Server className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">Node</p>
                <p className="text-sm text-muted-foreground">
                  {project.node_name || `Node ${project.node_id}`}
                  {project.node_ip && (
                    <span className="ml-2">({project.node_ip})</span>
                  )}
                </p>
              </div>
            </div>

            <Separator />

            {/* Path on disk */}
            <div className="flex items-start gap-3">
              <FolderOpen className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">Path on disk</p>
                <p className="text-sm text-muted-foreground font-mono bg-muted px-2 py-1 rounded break-all mt-1">
                  {project.path}
                </p>
              </div>
            </div>

            <Separator />

            {/* Timestamps */}
            <div className="flex items-start gap-3">
              <Clock className="h-5 w-5 mt-0.5 text-muted-foreground" />
              <div className="space-y-1">
                <p className="text-sm font-medium">First discovered</p>
                <p className="text-sm text-muted-foreground">
                  {format(new Date(project.discovered_at), "PPP")}
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 pl-8">
              <div className="space-y-1">
                <p className="text-sm font-medium">Last synced</p>
                <p className="text-sm text-muted-foreground">
                  {formatDistanceToNow(new Date(project.updated_at), {
                    addSuffix: true,
                  })}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Commands */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Terminal className="h-5 w-5" />
              Commands
            </CardTitle>
            <CardDescription>
              Runnable commands defined in config.vpspilot.json
            </CardDescription>
          </CardHeader>
          <CardContent>
            {project.commands && project.commands.length > 0 ? (
              <ul className="space-y-2">
                {project.commands.map((cmd, i) => (
                  <li
                    key={i}
                    className="flex items-center justify-between gap-3 p-3 rounded-md border bg-muted/40"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">{cmd.name}</p>
                      <p className="text-xs text-muted-foreground font-mono truncate">
                        {cmd.command}
                      </p>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled
                      title="Coming soon"
                    >
                      Run
                    </Button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground py-4 text-center">
                No commands defined
              </p>
            )}
          </CardContent>
        </Card>

        {/* Logs */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ScrollText className="h-5 w-5" />
              Log Paths
            </CardTitle>
            <CardDescription>
              Log directories configured for this project
            </CardDescription>
          </CardHeader>
          <CardContent>
            {project.logs && project.logs.length > 0 ? (
              <ul className="space-y-2">
                {project.logs.map((logPath, i) => (
                  <li
                    key={i}
                    className="flex items-center gap-2 text-sm font-mono bg-muted px-3 py-2 rounded"
                  >
                    <ScrollText className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    <span className="break-all">{logPath}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground py-4 text-center">
                No log paths configured
              </p>
            )}
          </CardContent>
        </Card>

        {/* Backups */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Archive className="h-5 w-5" />
              Backup Config
            </CardTitle>
            <CardDescription>
              Backup settings from config.vpspilot.json
            </CardDescription>
          </CardHeader>
          <CardContent>
            {project.backups ? (
              <div className="space-y-4">
                {project.backups.env_file && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
                      Env file
                    </p>
                    <p className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {project.backups.env_file}
                    </p>
                  </div>
                )}

                {project.backups.zip_file_name && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
                      Output archive
                    </p>
                    <p className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {project.backups.zip_file_name}.zip
                    </p>
                  </div>
                )}

                {project.backups.database && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1 flex items-center gap-1">
                      <Database className="h-3.5 w-3.5" />
                      Database
                    </p>
                    <div className="text-sm bg-muted rounded p-2 space-y-1 font-mono">
                      <p>connection: {project.backups.database.connection}</p>
                      <p>host: {project.backups.database.host}</p>
                      <p>port: {project.backups.database.port}</p>
                      <p>db: {project.backups.database.database_name}</p>
                    </div>
                  </div>
                )}

                {project.backups.dir && project.backups.dir.length > 0 && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
                      Directories
                    </p>
                    <ul className="space-y-1">
                      {project.backups.dir.map((d, i) => (
                        <li
                          key={i}
                          className="text-sm font-mono bg-muted px-2 py-1 rounded break-all"
                        >
                          {d}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground py-4 text-center">
                No backup config defined
              </p>
            )}
          </CardContent>
        </Card>
      </div>

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
