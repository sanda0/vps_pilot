import { ReactNode, useState } from "react";
import { Link } from "react-router";
import { format, formatDistanceToNow } from "date-fns";
import { Check, Clock, Copy, FolderOpen, RefreshCw, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Project } from "@/types/project";

function Fact({ icon, label, children }: { icon: ReactNode; label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 items-start gap-3">
      <span className="mt-0.5 text-muted-foreground">{icon}</span>
      <div className="min-w-0 flex-1">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
        <div className="mt-1 text-sm">{children}</div>
      </div>
    </div>
  );
}

// Where a project lives: its node, its directory, and when the agent found
// and last reported it.
export function ProjectInfo({ project }: { project: Project }) {
  const [copied, setCopied] = useState(false);
  const copyPath = () => {
    navigator.clipboard.writeText(project.path).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };
  const synced = new Date(project.updated_at);

  return (
    <Card>
      <CardContent className="grid gap-6 pt-6 sm:grid-cols-2 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)_auto_auto]">
        <Fact icon={<Server className="h-4 w-4" />} label="Node">
          <Link to={`/nodes/${project.node_id}`} className="font-medium hover:underline">
            {project.node_name || `Node ${project.node_id}`}
          </Link>
          {project.node_ip && <p className="font-mono text-xs text-muted-foreground">{project.node_ip}</p>}
        </Fact>
        <Fact icon={<FolderOpen className="h-4 w-4" />} label="Path on disk">
          <div className="flex items-start gap-1">
            <code className="min-w-0 break-all rounded bg-muted px-2 py-1 font-mono text-xs">{project.path}</code>
            <Button
              size="icon"
              variant="ghost"
              className="h-7 w-7 shrink-0"
              onClick={copyPath}
              title="Copy the path"
            >
              {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            </Button>
          </div>
        </Fact>
        <Fact icon={<Clock className="h-4 w-4" />} label="First discovered">
          {format(new Date(project.discovered_at), "PPP")}
        </Fact>
        <Fact icon={<RefreshCw className="h-4 w-4" />} label="Last synced">
          <span title={format(synced, "PPpp")}>{formatDistanceToNow(synced, { addSuffix: true })}</span>
        </Fact>
      </CardContent>
    </Card>
  );
}
