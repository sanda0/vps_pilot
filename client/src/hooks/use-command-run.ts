import { useEffect, useState } from "react";
import { commandRunsApi } from "@/lib/api";
import { CommandOutput, CommandRun, CommandRunMessage } from "@/types/command_run";

const MAX_RECONNECTS = 5;
const FLUSH_MS = 50;

// Follows a run over its WebSocket: the run, its output in seq order, and
// the final run. The hub replays all stored output on every connection, so
// chunks already seen are skipped. Reconnects while the run is running.
export function useCommandRun(runId: string | null) {
  const [run, setRun] = useState<CommandRun | null>(null);
  const [output, setOutput] = useState<CommandOutput[]>([]);

  useEffect(() => {
    setRun(null);
    setOutput([]);
    if (!runId) return;

    let ws: WebSocket | null = null;
    let stopped = false;
    let running = true;
    let lastSeq = -1;
    let reconnects = 0;
    let retry: ReturnType<typeof setTimeout> | undefined;
    // Output is added in batches, so a chatty command doesn't re-render
    // the page for every chunk.
    let pending: CommandOutput[] = [];
    let flush: ReturnType<typeof setTimeout> | undefined;

    const connect = () => {
      ws = new WebSocket(commandRunsApi.socketUrl(runId));
      ws.onmessage = (event) => {
        const msg = JSON.parse(event.data) as CommandRunMessage;
        if ("run" in msg) {
          running = msg.run.status === "running";
          setRun(msg.run);
          return;
        }
        if (msg.output.seq <= lastSeq) return;
        lastSeq = msg.output.seq;
        pending.push(msg.output);
        flush ??= setTimeout(() => {
          const batch = pending;
          pending = [];
          flush = undefined;
          setOutput((prev) => prev.concat(batch));
        }, FLUSH_MS);
      };
      ws.onopen = () => {
        reconnects = 0;
      };
      ws.onclose = () => {
        if (stopped || !running || reconnects >= MAX_RECONNECTS) return;
        reconnects++;
        retry = setTimeout(connect, 1000 * reconnects);
      };
    };
    connect();

    return () => {
      stopped = true;
      clearTimeout(retry);
      clearTimeout(flush);
      ws?.close();
    };
  }, [runId]);

  return { run, output };
}
