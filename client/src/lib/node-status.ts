import { NodeStatus } from "@/types/node_type"

// "Online", "Offline · last seen 5 min ago" or "Offline · never connected".
export function nodeStatusText(status: NodeStatus, now = Date.now()) {
  if (status.online) return "Online"
  if (!status.last_seen) return "Offline · never connected"
  const secs = Math.max(0, Math.floor(now / 1000 - status.last_seen))
  let ago: string
  if (secs < 60) ago = "just now"
  else if (secs < 3600) ago = `${Math.floor(secs / 60)} min ago`
  else if (secs < 86400) ago = `${Math.floor(secs / 3600)} h ago`
  else ago = `${Math.floor(secs / 86400)} d ago`
  return `Offline · last seen ${ago}`
}
