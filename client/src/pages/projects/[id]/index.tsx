import { useState, useEffect, useCallback } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { apiErrorMessage, commandRunsApi, projectsApi } from "@/lib/api";
import { Project } from "@/types/project";
import { CommandRun } from "@/types/command_run";
import { CommandRunPanel } from "@/components/command-run-panel";
import { CommandRunHistory } from "@/components/command-run-history";
import { ProjectLogs } from "@/components/project-logs";
import { ProjectBackups } from "@/components/project-backups";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
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
  Archive,
  Clock,
  Loader2,
  Play,
  History,
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
  const [runs, setRuns] = useState<CommandRun[]>([]);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [confirmCommand, setConfirmCommand] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const { toast } = useToast();
  const navigate = useNavigate();

  useEffect(() => {
    if (id) fetchProject();
  }, [id]);

  const fetchRuns = useCallback(async () => {
    if (!id) return;
    try {
      const list = await commandRunsApi.list(id);
      setRuns(list);
      // Show the latest run on first load, e.g. one still running.
      setSelectedRunId((current) => current ?? list[0]?.id ?? null);
    } catch {
      // The history is secondary; the page still works without it.
    }
  }, [id]);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  const startRun = async () => {
    if (!id || !confirmCommand) return;
    setStarting(true);
    try {
      const run = await commandRunsApi.start(id, confirmCommand);
      setSelectedRunId(run.id);
      fetchRuns();
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Couldn't run the command",
        description: apiErrorMessage(err, "The hub refused to start it"),
      });
    } finally {
      setStarting(false);
      setConfirmCommand(null);
    }
  };

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
                      onClick={() => setConfirmCommand(cmd.name)}
                    >
                      <Play className="mr-1 h-3 w-3" />
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
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ScrollText className="h-5 w-5" />
              Logs
            </CardTitle>
            <CardDescription>
              {project.logs && project.logs.length > 0 ? (
                <>
                  Files in{" "}
                  {project.logs.map((logPath, i) => (
                    <span key={logPath}>
                      {i > 0 && ", "}
                      <code className="font-mono">{logPath}</code>
                    </span>
                  ))}
                  , read as the owner of config.vpspilot.json
                </>
              ) : (
                "Log folders configured for this project"
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {project.logs && project.logs.length > 0 ? (
              <ProjectLogs projectId={project.id} />
            ) : (
              <p className="text-sm text-muted-foreground py-4 text-center">
                No log paths configured
              </p>
            )}
          </CardContent>
        </Card>

        {/* Backups */}
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Archive className="h-5 w-5" />
              Backups
            </CardTitle>
            <CardDescription>
              Zips of what config.vpspilot.json declares, made by the node's agent
            </CardDescription>
          </CardHeader>
          <CardContent>
            {project.backups ? (
              <ProjectBackups project={project} backups={project.backups} />
            ) : (
              <p className="text-sm text-muted-foreground py-4 text-center">
                No backup config defined
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Runs */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            Runs
          </CardTitle>
          <CardDescription>
            Commands run from the dashboard, with their output
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {selectedRunId ? (
            <CommandRunPanel runId={selectedRunId} onFinished={fetchRuns} />
          ) : (
            <p className="text-sm text-muted-foreground">
              Run a command to see its output here.
            </p>
          )}
          <CommandRunHistory
            runs={runs}
            selectedId={selectedRunId}
            onSelect={setSelectedRunId}
          />
        </CardContent>
      </Card>

      <AlertDialog
        open={confirmCommand !== null}
        onOpenChange={(open) => !open && setConfirmCommand(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Run {confirmCommand}?</AlertDialogTitle>
            <AlertDialogDescription>
              Runs{" "}
              <code className="font-mono">
                {project.commands.find((c) => c.name === confirmCommand)?.command}
              </code>{" "}
              in {project.path} on {project.node_name || `node ${project.node_id}`}, as the
              owner of its config.vpspilot.json.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={starting}>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={startRun} disabled={starting}>
              {starting ? "Starting..." : "Run"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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
