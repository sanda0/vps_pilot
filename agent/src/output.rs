//! Turns a command's stdout and stderr into `command_output` messages:
//! UTF-8 text in chunks of at most `chunk` bytes, numbered across both
//! streams, and nothing after `max` bytes. See "Output" in docs/PROTOCOL.md.

use tokio::sync::mpsc;

use crate::protocol::{CommandOutput, MSG_COMMAND_OUTPUT, Msg, Stream};

pub struct Emitter {
    run_id: String,
    out: mpsc::Sender<Msg>,
    chunk: usize,
    max: usize,
    seq: u64,
    sent: usize,
    pub truncated: bool,
    /// Bytes of a character split across reads, per stream.
    pending: [Vec<u8>; 2],
}

impl Emitter {
    pub fn new(run_id: &str, out: mpsc::Sender<Msg>, chunk: usize, max: usize) -> Self {
        Emitter {
            run_id: run_id.to_string(),
            out,
            chunk,
            max,
            seq: 0,
            sent: 0,
            truncated: false,
            pending: [Vec::new(), Vec::new()],
        }
    }

    /// Sends what was read from `stream`. With `eof`, also sends any
    /// incomplete character left over (as U+FFFD).
    pub async fn push(&mut self, stream: Stream, bytes: &[u8], eof: bool) {
        let pending = &mut self.pending[stream as usize];
        pending.extend_from_slice(bytes);
        let keep = if eof { 0 } else { incomplete_tail(pending) };
        let complete = pending.len() - keep;
        let text = String::from_utf8_lossy(&pending[..complete]).into_owned();
        pending.drain(..complete);

        for piece in pieces(&text, self.chunk) {
            if self.sent >= self.max {
                self.truncated = true;
                return;
            }
            self.sent += piece.len();
            let data = CommandOutput {
                run_id: self.run_id.clone(),
                seq: self.seq,
                stream,
                data: piece.to_string(),
            };
            self.seq += 1;
            let Ok(msg) = Msg::new(MSG_COMMAND_OUTPUT).with_data(&data) else {
                continue;
            };
            // A closed channel means the session is over; the run is
            // being torn down with it.
            let _ = self.out.send(msg).await;
        }
    }
}

/// Length of an incomplete UTF-8 character at the end of `buf`, which the
/// next read will complete.
pub fn incomplete_tail(buf: &[u8]) -> usize {
    for back in 1..=buf.len().min(3) {
        let b = buf[buf.len() - back];
        if b & 0xC0 == 0x80 {
            continue;
        }
        let need = match b {
            0xC2..=0xDF => 2,
            0xE0..=0xEF => 3,
            0xF0..=0xF4 => 4,
            _ => return 0,
        };
        return if need > back { back } else { 0 };
    }
    0
}

/// Splits `s` into pieces of at most `max` bytes, on character boundaries.
fn pieces(s: &str, max: usize) -> Vec<&str> {
    let mut out = Vec::new();
    let mut rest = s;
    while !rest.is_empty() {
        let mut end = rest.len().min(max);
        while !rest.is_char_boundary(end) {
            end -= 1;
        }
        let (piece, tail) = rest.split_at(end);
        out.push(piece);
        rest = tail;
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    fn drain(rx: &mut mpsc::Receiver<Msg>) -> Vec<CommandOutput> {
        let mut got = Vec::new();
        while let Ok(msg) = rx.try_recv() {
            let d = msg.data.unwrap();
            got.push(CommandOutput {
                run_id: d["run_id"].as_str().unwrap().into(),
                seq: d["seq"].as_u64().unwrap(),
                stream: if d["stream"] == "stdout" {
                    Stream::Stdout
                } else {
                    Stream::Stderr
                },
                data: d["data"].as_str().unwrap().into(),
            });
        }
        got
    }

    #[tokio::test]
    async fn keeps_characters_split_across_reads() {
        let (tx, mut rx) = mpsc::channel(16);
        let mut e = Emitter::new("r1", tx, 1024, 1 << 20);
        let euro = "€".as_bytes(); // 3 bytes
        e.push(Stream::Stdout, &[b'a', euro[0]], false).await;
        e.push(Stream::Stderr, b"err", false).await;
        e.push(Stream::Stdout, &euro[1..], false).await;
        e.push(Stream::Stdout, &[0xE2], true).await;

        let got = drain(&mut rx);
        let texts: Vec<_> = got.iter().map(|o| (o.seq, o.data.as_str())).collect();
        assert_eq!(texts, [(0, "a"), (1, "err"), (2, "€"), (3, "\u{FFFD}")]);
        assert_eq!(got[1].stream, Stream::Stderr);
    }

    #[tokio::test]
    async fn splits_into_chunks_and_stops_at_max() {
        let (tx, mut rx) = mpsc::channel(16);
        let mut e = Emitter::new("r1", tx, 4, 8);
        e.push(Stream::Stdout, "ab€cdefghij".as_bytes(), false)
            .await;

        let got: Vec<_> = drain(&mut rx).into_iter().map(|o| o.data).collect();
        assert_eq!(got, ["ab", "€c", "defg"]);
        assert!(e.truncated);
    }

    #[test]
    fn incomplete_tail_lengths() {
        assert_eq!(incomplete_tail(b"abc"), 0);
        assert_eq!(incomplete_tail("é".as_bytes()), 0);
        assert_eq!(incomplete_tail(&"😀".as_bytes()[..3]), 3);
        assert_eq!(incomplete_tail(&[b'a', 0xFF]), 0);
    }
}
