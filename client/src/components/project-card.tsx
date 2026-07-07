import { Project } from "@/types/project";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Eye, Trash2, Server, FolderOpen, Terminal } from "lucide-react";
import { Link } from "react-router";
import { formatDistanceToNow } from "date-fns";

interface ProjectCardProps {
  project: Project;
  onDelete?: (id: string) => void;
}

export function ProjectCard({ project, onDelete }: ProjectCardProps) {
  return (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <CardTitle className="text-xl leading-tight">
            {project.name}
          </CardTitle>
        </div>

        {/* Tech stack badges */}
        {project.tech && project.tech.length > 0 && (
          <div className="flex flex-wrap gap-1 pt-1">
            {project.tech.map((t) => (
              <Badge key={t} variant="secondary" className="text-xs">
                {t}
              </Badge>
            ))}
          </div>
        )}
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Node */}
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Server className="h-4 w-4 shrink-0" />
          <span className="font-medium">
            {project.node_name || `Node ${project.node_id}`}
          </span>
          {project.node_ip && (
            <span className="text-xs">({project.node_ip})</span>
          )}
        </div>

        {/* Path on disk */}
        <div className="flex items-start gap-2 text-sm text-muted-foreground">
          <FolderOpen className="h-4 w-4 shrink-0 mt-0.5" />
          <span className="font-mono text-xs bg-muted px-2 py-1 rounded break-all">
            {project.path}
          </span>
        </div>

        {/* Commands count */}
        {project.commands && project.commands.length > 0 && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Terminal className="h-4 w-4 shrink-0" />
            <span>
              {project.commands.length} command
              {project.commands.length !== 1 ? "s" : ""} available
            </span>
          </div>
        )}

        {/* Last synced */}
        <div className="text-xs text-muted-foreground pt-1 border-t">
          Last synced{" "}
          {formatDistanceToNow(new Date(project.updated_at), {
            addSuffix: true,
          })}
        </div>
      </CardContent>

      <CardFooter className="flex gap-2">
        <Button asChild variant="outline" size="sm" className="flex-1">
          <Link to={`/projects/${project.id}`}>
            <Eye className="h-4 w-4 mr-1" />
            View
          </Link>
        </Button>
        {onDelete && (
          <Button
            variant="destructive"
            size="sm"
            onClick={() => onDelete(project.id)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
      </CardFooter>
    </Card>
  );
}
