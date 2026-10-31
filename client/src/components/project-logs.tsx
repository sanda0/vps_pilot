import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { ChevronUp, Loader2, RefreshCw, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useLogFile } from "@/hooks/use-log-file";
import { apiErrorMessage, logsApi } from "@/lib/api";
import { formatBytes } from "@/lib/utils";
import { LogEnd, LogFile } from "@/types/log";

// The agent lists compressed files but can't open them.
const COMPRESSED = /\.(gz|bz2|xz|zst|zip)$/;

function endMessage(end: LogEnd): string {
  switch (end.reason) {
    case "limit":
      return "Stopped following after 30 minutes.";
    case "closed":
    case "done":
      return "Stopped following.";
    default:
      return end.error || "Reading the file failed.";
  }
}

// Lists the files in a project's log folders and follows the one picked.
export function ProjectLogs({ projectId }: { projectId: string }) {
  const [files, setFiles] = useState<LogFile[]>([]);
  const [listError, setListError] = useState("");
  const [listing, setListing] = useState(true);
  const [file, setFile] = useState<string | null>(null);

  const fetchFiles = useCallback(async () => {
    setListing(true);
    try {
      const list = await logsApi.list(projectId);
      setFiles(list.files);
      setListError(list.error);
      // Start with the most recently written file.
      const readable = list.files.filter((f) => !COMPRESSED.test(f.path));
      setFile((current) => current ?? readable.sort((a, b) => b.modified - a.modified)[0]?.path ?? null);
    } catch (err) {
      setListError(apiErrorMessage(err, "Couldn't list the log files"));
    } finally {
      setListing(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchFiles();
  }, [fetchFiles]);

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1">
          <Select value={file ?? undefined} onValueChange={setFile} disabled={files.length === 0}>
            <SelectTrigger className="font-mono text-xs">
              <SelectValue placeholder={listing ? "Loading files…" : "No log files"} />
            </SelectTrigger>
            <SelectContent>
              {files.map((f) => (
                <SelectItem key={f.path} value={f.path} disabled={COMPRESSED.test(f.path)} className="font-mono text-xs">
                  {f.path}{" "}
                  <span className="text-muted-foreground">
                    {formatBytes(f.size)} · {formatDistanceToNow(new Date(f.modified * 1000), { addSuffix: true })}
                    {COMPRESSED.test(f.path) && " · compressed"}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button size="icon" variant="outline" onClick={fetchFiles} disabled={listing} title="Refresh the file list">
          <RefreshCw className={listing ? "h-4 w-4 animate-spin" : "h-4 w-4"} />
        </Button>
      </div>
      {listError && <p className="text-sm text-destructive">{listError}</p>}
      {file && <LogView key={file} projectId={projectId} file={file} />}
    </div>
  );
}

function LogView({ projectId, file }: { projectId: string; file: string }) {
  const { chunks, end, earlier, loadEarlier, reload, start } = useLogFile(projectId, file);
  const outputRef = useRef<HTMLPreElement>(null);
  const followRef = useRef(true);
  const heightRef = useRef(0);
  const startRef = useRef(start);

  // Keep the newest lines in view unless the user scrolled up, and keep the
  // same lines in view when earlier ones are added above them.
  useLayoutEffect(() => {
    const el = outputRef.current;
    if (!el) return;
    if (start < startRef.current) {
      el.scrollTop += el.scrollHeight - heightRef.current;
    } else if (followRef.current) {
      el.scrollTop = el.scrollHeight;
    }
    startRef.current = start;
    heightRef.current = el.scrollHeight;
  }, [chunks, start]);

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
        {start > 0 ? (
          <Button size="sm" variant="ghost" onClick={loadEarlier} disabled={earlier.loading}>
            {earlier.loading ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : <ChevronUp className="mr-1 h-3 w-3" />}
            Load earlier
          </Button>
        ) : (
          <span>{chunks.length > 0 && "Start of file"}</span>
        )}
        {end ? (
          <span className="flex items-center gap-2">
            {endMessage(end)}
            <Button size="sm" variant="outline" onClick={reload}>
              <RotateCcw className="mr-1 h-3 w-3" />
              Follow again
            </Button>
          </span>
        ) : (
          <span className="flex items-center gap-1">
            <span className="h-2 w-2 animate-pulse rounded-full bg-green-500" />
            Following
          </span>
        )}
      </div>
      {earlier.error && <p className="text-xs text-destructive">{earlier.error}</p>}
      <pre
        ref={outputRef}
        onScroll={(e) => {
          const el = e.currentTarget;
          followRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
        }}
        className="h-96 overflow-auto whitespace-pre-wrap break-all border bg-muted p-3 font-mono text-xs"
      >
        {chunks.length === 0 ? (
          <span className="text-muted-foreground">{end ? "No lines" : "Waiting for lines…"}</span>
        ) : (
          chunks.map((c) => <span key={c.offset}>{c.data}</span>)
        )}
      </pre>
    </div>
  );
}
