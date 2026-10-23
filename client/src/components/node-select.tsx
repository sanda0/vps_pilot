import { useEffect, useState } from "react";
import { nodesApi } from "@/lib/api";
import { NodeOption } from "@/types/project_scan";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface NodeSelectProps {
  value: number | null;
  onChange: (nodeId: number) => void;
}

// Picks one of the hub's nodes. Selects the first node once loaded if none is chosen.
export function NodeSelect({ value, onChange }: NodeSelectProps) {
  const [nodes, setNodes] = useState<NodeOption[]>([]);

  useEffect(() => {
    nodesApi
      .list()
      .then((list) => {
        setNodes(list);
        if (value === null && list.length > 0) onChange(list[0].id);
      })
      .catch((err) => console.error("Failed to load nodes:", err));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Select
      value={value === null ? undefined : String(value)}
      onValueChange={(v) => onChange(Number(v))}
      disabled={nodes.length === 0}
    >
      <SelectTrigger>
        <SelectValue placeholder={nodes.length === 0 ? "No nodes yet" : "Select a node"} />
      </SelectTrigger>
      <SelectContent>
        {nodes.map((n) => (
          <SelectItem key={n.id} value={String(n.id)}>
            {n.name || `Node ${n.id}`} <span className="text-muted-foreground">({n.ip})</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
