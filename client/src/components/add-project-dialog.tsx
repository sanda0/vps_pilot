import { useState } from "react";
import { apiErrorMessage, projectScanApi } from "@/lib/api";
import { useToast } from "@/hooks/use-toast";
import { NodeSelect } from "@/components/node-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface AddProjectDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// Adds a project directory by hand. The agent still needs a
// config.vpspilot.json there and only accepts paths inside its --projects-dir.
export function AddProjectDialog({ open, onOpenChange }: AddProjectDialogProps) {
  const [nodeId, setNodeId] = useState<number | null>(null);
  const [path, setPath] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { toast } = useToast();

  const close = (next: boolean) => {
    if (!next) {
      setPath("");
      setError(null);
    }
    onOpenChange(next);
  };

  const save = async () => {
    if (nodeId === null) return;
    setSaving(true);
    setError(null);
    try {
      const added = await projectScanApi.addPath(nodeId, path);
      toast({
        title: "Project path added",
        description: `The agent will look for config.vpspilot.json in ${added.path}. Problems show under Scan settings.`,
      });
      close(false);
    } catch (err) {
      setError(apiErrorMessage(err, "Failed to add the project path"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Add project</DialogTitle>
          <DialogDescription>
            For a project the automatic search doesn't reach, e.g. one nested deeper than 4 levels. The
            directory needs a <code className="text-xs bg-muted px-1 py-0.5">config.vpspilot.json</code>, and
            must be inside the agent's project directories unless the agent runs with{" "}
            <code className="text-xs bg-muted px-1 py-0.5">--allow-any-project-path</code>.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-2">
          <div className="grid gap-2">
            <Label>Node</Label>
            <NodeSelect value={nodeId} onChange={setNodeId} />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="project-path">Project directory</Label>
            <Input
              id="project-path"
              placeholder="/var/www/clients/acme/app"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && save()}
            />
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => close(false)}>
            Cancel
          </Button>
          <Button onClick={save} disabled={saving || nodeId === null || path.trim() === ""}>
            {saving ? "Adding…" : "Add"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
