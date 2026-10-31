import { useCallback, useEffect, useRef, useState } from "react";
import { logsApi } from "@/lib/api";
import { LogChunk, LogEnd, LogMessage } from "@/types/log";

const FLUSH_MS = 50;
const EARLIER_BYTES = 256 * 1024;
// Older chunks are dropped past this; "load earlier" brings them back.
const MAX_CHARS = 2 * 1024 * 1024;

const NO_SOCKET: LogEnd = { reason: "error", error: "Couldn't open the file. Is the node online?" };

// Keeps the newest chunks within MAX_CHARS, and always the last one.
function trim(chunks: LogChunk[]): LogChunk[] {
  let total = 0;
  let i = chunks.length;
  while (i > 0 && total + chunks[i - 1].data.length <= MAX_CHARS) {
    total += chunks[--i].data.length;
  }
  return chunks.slice(Math.min(i, chunks.length - 1));
}

// Opens one stream and passes its chunks on, then its end. Returns a
// function that closes it.
function stream(url: string, onChunk: (c: LogChunk) => void, onEnd: (e: LogEnd) => void) {
  const ws = new WebSocket(url);
  let ended = false;
  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data) as LogMessage;
    if ("data" in msg) {
      onChunk(msg.data);
    } else {
      ended = true;
      onEnd(msg.end);
    }
  };
  ws.onclose = () => {
    if (!ended) onEnd(NO_SOCKET);
  };
  return () => {
    ended = true;
    ws.close();
  };
}

// Follows the end of a log file, and loads earlier parts on request.
// A rotated file starts again from what the agent sends next.
export function useLogFile(projectId: string, file: string | null) {
  const [chunks, setChunks] = useState<LogChunk[]>([]);
  const [end, setEnd] = useState<LogEnd | null>(null);
  const [earlier, setEarlier] = useState<{ loading: boolean; error: string }>({ loading: false, error: "" });
  const [generation, setGeneration] = useState(0);
  const closeEarlier = useRef<(() => void) | null>(null);

  useEffect(() => {
    setChunks([]);
    setEnd(null);
    setEarlier({ loading: false, error: "" });
    if (!file) return;

    // Chunks are added in batches, so a busy log doesn't re-render the page
    // for every one.
    let pending: LogChunk[] = [];
    let flush: ReturnType<typeof setTimeout> | undefined;
    const close = stream(
      logsApi.socketUrl(projectId, file, { follow: true }),
      (chunk) => {
        if (chunk.rotated) pending = [];
        pending.push(chunk);
        flush ??= setTimeout(() => {
          const batch = pending;
          pending = [];
          flush = undefined;
          setChunks((prev) => trim(batch[0].rotated ? batch : prev.concat(batch)));
        }, FLUSH_MS);
      },
      setEnd,
    );
    return () => {
      clearTimeout(flush);
      close();
      closeEarlier.current?.();
    };
  }, [projectId, file, generation]);

  const loadEarlier = useCallback(() => {
    const before = chunks[0]?.offset;
    if (!file || !before || earlier.loading) return;
    setEarlier({ loading: true, error: "" });
    const got: LogChunk[] = [];
    closeEarlier.current = stream(
      logsApi.socketUrl(projectId, file, { before, maxBytes: EARLIER_BYTES }),
      (chunk) => got.push(chunk),
      (e) => {
        closeEarlier.current = null;
        if (e.reason !== "done") {
          setEarlier({ loading: false, error: e.error || "Couldn't load earlier lines" });
          return;
        }
        setChunks((prev) => (prev[0]?.offset === before ? got.concat(prev) : prev));
        setEarlier({ loading: false, error: "" });
      },
    );
  }, [projectId, file, chunks, earlier.loading]);

  // Opens the file again from its end, e.g. after following stopped.
  const reload = useCallback(() => setGeneration((g) => g + 1), []);

  return { chunks, end, earlier, loadEarlier, reload, start: chunks[0]?.offset ?? 0 };
}
