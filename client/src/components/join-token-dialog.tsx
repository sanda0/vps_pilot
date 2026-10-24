import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { nodesApi } from "@/lib/api";
import { JoinToken } from "@/types/node_type";
import { format } from "date-fns";
import { Check, Copy, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";

interface JoinTokenDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  // When set, the token re-enrolls this node instead of adding a new one.
  nodeId?: number;
}

export function JoinTokenDialog({ open, onOpenChange, nodeId }: JoinTokenDialogProps) {
  const [token, setToken] = useState<JoinToken | null>(null);
  const [error, setError] = useState<string | null>(null);
  // Which command was copied last, to show its check mark.
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setToken(null);
      setError(null);
      setCopied(null);
      return;
    }
    nodesApi
      .createJoinToken(nodeId)
      .then(setToken)
      .catch((err) => setError(err.response?.data?.error || "Failed to create join token"));
  }, [open, nodeId]);

  const copy = (command: string) => {
    navigator.clipboard.writeText(command).then(() => setCopied(command));
  };

  const commandBox = (command: string) => (
    <div className="relative">
      <pre className="p-3 pr-12 overflow-x-auto font-mono text-xs rounded-md bg-muted whitespace-pre-wrap break-all">
        {command}
      </pre>
      <Button size="icon" variant="ghost" className="absolute top-1 right-1" onClick={() => copy(command)}>
        {copied === command ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
      </Button>
    </div>
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>{nodeId ? "Re-enroll agent" : "Add node"}</DialogTitle>
          <DialogDescription>
            {nodeId
              ? "Run this on the server to install a fresh agent for this node. Its current key stops working once the new agent enrolls."
              : "Run this on the server. It installs the VPS Pilot agent as a systemd service and enrolls it. The token works once."}
          </DialogDescription>
        </DialogHeader>

        {error && <p className="text-sm text-destructive">{error}</p>}

        {!token && !error && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" /> Creating join token…
          </div>
        )}

        {token && (
          <div className="grid gap-3">
            {commandBox(token.install_command)}
            <p className="text-xs text-muted-foreground">Agent already installed? Run it with the token instead:</p>
            {commandBox(token.command)}
            <p className="text-xs text-muted-foreground">
              Expires {format(new Date(token.expires_at * 1000), "PPp")}. It won't be shown again.
            </p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
