//! Where a command's stdout and stderr go. `Emitter` turns them into
//! `command_output` messages: UTF-8 text in chunks of at most `chunk` bytes,
//! numbered across both streams, and nothing after `max` bytes (see "Output"
//! in docs/PROTOCOL.md). `Tail` keeps only the last `max` bytes of both, for
//! a cron job's result (see "Cron jobs").

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

/// The last `max` bytes of stdout and stderr together, in arrival order.
pub struct Tail {
    max: usize,
    buf: Vec<u8>,
    pending: [Vec<u8>; 2],
    pub truncated: bool,
}

impl Tail {
    pub fn new(max: usize) -> Self {
        Tail {
            max,
            buf: Vec::new(),
            pending: [Vec::new(), Vec::new()],
            truncated: false,
        }
    }

    /// Adds what was read from `stream`. A character split across reads is
    /// held back until it's complete, so the streams don't garble it.
    pub fn push(&mut self, stream: Stream, bytes: &[u8], eof: bool) {
        let pending = &mut self.pending[stream as usize];
        pending.extend_from_slice(bytes);
        let keep = if eof { 0 } else { incomplete_tail(pending) };
        let complete = pending.len() - keep;
        self.buf.extend(pending.drain(..complete));
        // Cut now and then rather than on every read.
        if self.buf.len() > 2 * self.max {
            self.cut();
        }
    }

    /// The kept output as UTF-8, invalid bytes replaced by U+FFFD, and
    /// whether anything was dropped.
    pub fn finish(mut self) -> (String, bool) {
        for stream in [Stream::Stdout, Stream::Stderr] {
            self.push(stream, &[], true);
        }
        self.cut();
        (
            String::from_utf8_lossy(&self.buf).into_owned(),
            self.truncated,
        )
    }

    fn cut(&mut self) {
        if self.buf.len() <= self.max {
            return;
        }
        let mut start = self.buf.len() - self.max;
        // Start on a character, not in the middle of one.
        let limit = (start + 3).min(self.buf.len());
        while start < limit && self.buf[start] & 0xC0 == 0x80 {
            start += 1;
        }
        self.buf.drain(..start);
        self.truncated = true;
    }
}

/// Where `commands::run` sends output.
pub enum Sink {
    Stream(Emitter),
    Tail(Tail),
}

impl Sink {
    pub async fn push(&mut self, stream: Stream, bytes: &[u8], eof: bool) {
        match self {
            Sink::Stream(e) => e.push(stream, bytes, eof).await,
            Sink::Tail(t) => t.push(stream, bytes, eof),
        }
    }

    /// The kept output (empty when it was streamed), and whether any
    /// output was dropped.
    pub fn finish(self) -> (String, bool) {
        match self {
            Sink::Stream(e) => (String::new(), e.truncated),
            Sink::Tail(t) => t.finish(),
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
    fn tail_keeps_the_end_in_order() {
        let mut t = Tail::new(8);
        let euro = "€".as_bytes();
        t.push(Stream::Stdout, &[b'a', euro[0]], false);
        t.push(Stream::Stderr, b"E", false);
        t.push(Stream::Stdout, &euro[1..], false);
        assert!(!t.truncated);
        assert_eq!(t.finish(), ("aE€".into(), false));

        // The last 8 bytes start inside "€", so the cut moves past it.
        let mut t = Tail::new(8);
        t.push(Stream::Stdout, "0123456789€uvwxyz".as_bytes(), false);
        assert_eq!(t.finish(), ("uvwxyz".into(), true));
        let mut t = Tail::new(8);
        t.push(Stream::Stdout, b"0123456789abcdefghij", true);
        assert!(t.buf.len() <= 16);
        assert_eq!(t.finish(), ("cdefghij".into(), true));
    }

    #[test]
    fn incomplete_tail_lengths() {
        assert_eq!(incomplete_tail(b"abc"), 0);
        assert_eq!(incomplete_tail("é".as_bytes()), 0);
        assert_eq!(incomplete_tail(&"😀".as_bytes()[..3]), 3);
        assert_eq!(incomplete_tail(&[b'a', 0xFF]), 0);
    }
}
