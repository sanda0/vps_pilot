// A readable file in a project's log folders (GET /projects/:id/logs).
export interface LogFile {
  path: string; // relative to the project directory
  size: number; // bytes
  modified: number; // Unix seconds
}

export interface LogFiles {
  files: LogFile[];
  error: string; // the agent refused, or cut the list at 1000 files
}

export interface LogChunk {
  seq: number;
  offset: number; // byte offset in the file where data starts
  data: string;
  size: number; // the file's size when the chunk was read
  rotated: boolean; // the file was truncated or replaced; data starts it again
}

export type LogEndReason = "done" | "closed" | "rejected" | "error" | "limit";

export interface LogEnd {
  reason: LogEndReason;
  error: string;
}

// One message on /projects/:id/logs/ws.
export type LogMessage = { data: LogChunk } | { end: LogEnd };
