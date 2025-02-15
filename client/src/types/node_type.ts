
export interface NodeData {

  id: number;
  name: string;
  ip: string;
  cpus: number;
  total_memory: number;
}

export interface NodeSysInfo {
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