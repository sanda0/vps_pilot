import { AlertTriangle, FolderOpen, X } from "lucide-react";
import { ProjectScan } from "@/types/project_scan";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

interface ScanPathsAndErrorsProps {
  scan: ProjectScan;
  busy: boolean;
  onRemovePath: (pathId: number) => void;
}

// Shows a long path cut at the start ("…/clients/acme/app"), since the end is
// what tells paths apart. <bdi> keeps the slashes in order under rtl.
function PathText({ path, className = "" }: { path: string; className?: string }) {
  return (
    <code className={`truncate text-left [direction:rtl] ${className}`} title={path}>
      <bdi>{path}</bdi>
    </code>
  );
}

// A node's project directories added by hand, and the problems its agent
// reported in the last scan (broken configs, rejected paths).
export function ScanPathsAndErrors({ scan, busy, onRemovePath }: ScanPathsAndErrorsProps) {
  return (
    <>
      <div className="grid min-w-0 gap-2">
        <Label>Added by hand</Label>
        {scan.paths.length === 0 ? (
          <p className="text-sm text-muted-foreground">None. Use Add project to point the agent at a directory.</p>
        ) : (
          <ul className="min-w-0 divide-y border">
            {scan.paths.map((p) => (
              <li key={p.id} className="flex min-w-0 items-center gap-2 px-3 py-1.5 text-sm">
                <FolderOpen className="h-4 w-4 shrink-0 text-muted-foreground" />
                <PathText path={p.path} className="min-w-0 flex-1" />
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => onRemovePath(p.id)}
                  disabled={busy}
                  aria-label={`Remove ${p.path}`}
                >
                  <X className="h-4 w-4" />
                </Button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {scan.errors.length > 0 && (
        <div className="grid min-w-0 gap-2">
          <Label className="flex items-center gap-1.5 text-destructive">
            <AlertTriangle className="h-4 w-4" />
            Problems in the last scan ({scan.errors.length})
          </Label>
          <ul className="min-w-0 divide-y border border-destructive/40">
            {scan.errors.map((e) => (
              <li key={e.path} className="grid min-w-0 gap-0.5 px-3 py-1.5 text-sm">
                <PathText path={e.path} />
                <span className="text-muted-foreground">{e.error}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}
