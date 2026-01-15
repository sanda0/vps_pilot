import { Project } from '@/types/project';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { ProjectStatusBadge } from '@/components/project-status-badge';
import { Button } from '@/components/ui/button';
import { Eye, Pencil, Trash2, Server, FolderGit2, GitBranch } from 'lucide-react';
import { Link } from 'react-router';
import { formatDistanceToNow } from 'date-fns';

interface ProjectCardProps {
  project: Project;
  onDelete?: (id: string) => void;
}

export function ProjectCard({ project, onDelete }: ProjectCardProps) {
  return (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="text-xl">{project.name}</CardTitle>
            <CardDescription className="line-clamp-2">
              {project.description || 'No description'}
            </CardDescription>
          </div>
          <ProjectStatusBadge status={project.status} />
        </div>
      </CardHeader>
      
      <CardContent className="space-y-3">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Server className="h-4 w-4" />
          <span className="font-medium">{project.node_name || `Node ${project.node_id}`}</span>
          {project.node_ip && <span className="text-xs">({project.node_ip})</span>}
        </div>

        {project.repo_url && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <FolderGit2 className="h-4 w-4" />
            <span className="truncate">{project.repo_url}</span>
          </div>
        )}

        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <GitBranch className="h-4 w-4" />
          <span>{project.branch}</span>
        </div>

        <div className="text-sm text-muted-foreground">
          <span className="font-mono text-xs bg-muted px-2 py-1 rounded">
            {project.deploy_path}
          </span>
        </div>

        {project.last_deployed_at && (
          <div className="text-xs text-muted-foreground pt-2 border-t">
            Last deployed: {formatDistanceToNow(new Date(project.last_deployed_at), { addSuffix: true })}
          </div>
        )}
      </CardContent>

      <CardFooter className="flex gap-2">
        <Button asChild variant="outline" size="sm" className="flex-1">
          <Link to={`/projects/${project.id}`}>
            <Eye className="h-4 w-4 mr-1" />
            View
          </Link>
        </Button>
        <Button asChild variant="outline" size="sm" className="flex-1">
          <Link to={`/projects/${project.id}/edit`}>
            <Pencil className="h-4 w-4 mr-1" />
            Edit
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
