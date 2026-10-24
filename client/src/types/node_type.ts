
// Whether the node's agent is connected, from the hub's live connections.
export interface NodeStatus {
  online: boolean;
  last_seen: number; // unix seconds of the latest stats, 0 if never
}

export interface NodeData extends NodeStatus {

  id: number;
  name: string;
  ip: string;
  cpus: number;
  total_memory: number;
}

export interface NodeSysInfo extends NodeStatus {
  id: number;
  name: string;
  ip: string;
  os: string;
  platform: string;
  platform_version: string;
  kernel_version: string;
  cpus: number;
  total_memory: number;
}
export interface JoinToken {
  token: string;
  expires_at: number; // unix seconds
  command: string; // runs an agent that is already installed
  install_command: string; // installs the agent as a service and enrolls it
}
