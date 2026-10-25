//! Reads a window of a log file and, optionally, what is appended to it.
//! Runs in the log child process, as the config file's owner, and writes
//! one JSON `Chunk` per line for the agent to forward. See "Reading a file"
//! in docs/PROTOCOL.md.

use std::fs::{self, File};
use std::io::{self, ErrorKind, Read, Seek, SeekFrom, Write};
use std::os::unix::fs::MetadataExt;
use std::path::PathBuf;
use std::time::Duration;

use serde::{Deserialize, Serialize};

use crate::output::incomplete_tail;

pub const CHUNK: usize = 32 * 1024;
pub const MAX_WINDOW: u64 = 1 << 20;
/// Most bytes read per poll while following.
const MAX_PER_POLL: u64 = 1 << 20;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Chunk {
    pub offset: u64,
    pub data: String,
    pub size: u64,
    pub rotated: bool,
}

pub struct Request {
    /// Byte offset the window ends at; None for the end of the file.
    pub before: Option<u64>,
    pub max_bytes: u64,
    pub follow: bool,
    pub poll: Duration,
}

/// Sends the window, then, with `follow`, what is appended until writing
/// fails (the agent closed the pipe). `open` resolves the file; it runs
/// again whenever the file is truncated or replaced, so a rotated file must
/// still pass the checks in logfiles.
pub fn read(
    open: impl Fn() -> io::Result<PathBuf>,
    req: &Request,
    out: &mut impl Write,
) -> io::Result<()> {
    let path = open()?;
    let mut file = File::open(&path)?;
    let meta = file.metadata()?;
    let (size, mut ino) = (meta.len(), meta.ino());
    let end = req.before.unwrap_or(size).min(size);
    let start = end.saturating_sub(req.max_bytes.clamp(1, MAX_WINDOW));
    let mut buf = vec![0; (end - start) as usize];
    file.seek(SeekFrom::Start(start))?;
    file.read_exact(&mut buf)?;

    // Start at a line, unless that would drop the whole window.
    let mut skip = 0;
    if start > 0
        && let Some(nl) = buf.iter().position(|&b| b == b'\n')
    {
        skip = nl + 1;
    }
    let used = emit(
        out,
        &buf[skip..],
        start + skip as u64,
        size,
        false,
        !req.follow,
    )?;
    if !req.follow {
        return Ok(());
    }

    let mut pos = start + (skip + used) as u64;
    let mut rotated = false;
    loop {
        std::thread::sleep(req.poll);
        let now = match fs::metadata(&path) {
            Ok(m) => m,
            // Between rotating the old file away and creating the new one.
            Err(e) if e.kind() == ErrorKind::NotFound => continue,
            Err(e) => return Err(e),
        };
        if now.ino() != ino || now.len() < pos {
            file = File::open(open()?)?;
            ino = file.metadata()?.ino();
            (pos, rotated) = (0, true);
        }
        let len = file.metadata()?.len();
        if len <= pos {
            continue;
        }
        let mut buf = vec![0; (len - pos).min(MAX_PER_POLL) as usize];
        file.seek(SeekFrom::Start(pos))?;
        let n = file.read(&mut buf)?;
        let used = emit(out, &buf[..n], pos, len, rotated, false)?;
        if used > 0 {
            rotated = false;
        }
        pos += used as u64;
    }
}

/// Writes `bytes` (starting at `offset` in the file) as chunks of at most
/// CHUNK bytes cut on character boundaries. Unless `last`, an incomplete
/// character at the end is left for the next read. Returns the bytes used.
fn emit(
    out: &mut impl Write,
    bytes: &[u8],
    offset: u64,
    size: u64,
    rotated: bool,
    last: bool,
) -> io::Result<usize> {
    let len = bytes.len() - if last { 0 } else { incomplete_tail(bytes) };
    let mut at = 0;
    while at < len {
        let mut end = (at + CHUNK).min(len);
        while end < len && end > at + 1 && bytes[end] & 0xC0 == 0x80 {
            end -= 1;
        }
        let chunk = Chunk {
            offset: offset + at as u64,
            data: String::from_utf8_lossy(&bytes[at..end]).into_owned(),
            size,
            rotated: rotated && at == 0,
        };
        serde_json::to_writer(&mut *out, &chunk)?;
        out.write_all(b"\n")?;
        at = end;
    }
    out.flush()?;
    Ok(len)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::mpsc;

    fn request(before: Option<u64>, max_bytes: u64, follow: bool) -> Request {
        let poll = Duration::from_millis(20);
        Request {
            before,
            max_bytes,
            follow,
            poll,
        }
    }

    fn window(content: &[u8], before: Option<u64>, max_bytes: u64) -> Vec<Chunk> {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("app.log");
        fs::write(&path, content).unwrap();
        let mut out = Vec::new();
        read(
            || Ok(path.clone()),
            &request(before, max_bytes, false),
            &mut out,
        )
        .unwrap();
        out.split(|&b| b == b'\n')
            .filter(|l| !l.is_empty())
            .map(|l| serde_json::from_slice(l).unwrap())
            .collect()
    }

    #[test]
    fn window_starts_at_a_line_and_loads_earlier() {
        let log = b"aaa\nbbb\nccc\n";
        let last = window(log, None, 6);
        assert_eq!(
            (last[0].offset, last[0].data.as_str(), last[0].size),
            (8, "ccc\n", 12)
        );
        let earlier = window(log, Some(8), 100);
        assert_eq!(
            (earlier[0].offset, earlier[0].data.as_str()),
            (0, "aaa\nbbb\n")
        );
    }

    #[test]
    fn chunks_split_on_characters() {
        let mut log = vec![b'a'; CHUNK - 1];
        log.extend_from_slice("€ end".as_bytes());
        let chunks = window(&log, None, MAX_WINDOW);
        assert_eq!(chunks.len(), 2);
        assert_eq!(
            (chunks[0].data.len(), chunks[1].offset),
            (CHUNK - 1, CHUNK as u64 - 1)
        );
        assert_eq!(chunks[1].data, "€ end");
    }

    /// Forwards each written line to a channel; fails once it's dropped.
    struct Lines(mpsc::Sender<Chunk>, Vec<u8>);

    impl Write for Lines {
        fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
            self.1.extend_from_slice(buf);
            while let Some(nl) = self.1.iter().position(|&b| b == b'\n') {
                let line: Vec<u8> = self.1.drain(..=nl).collect();
                let chunk = serde_json::from_slice(&line).unwrap();
                self.0.send(chunk).map_err(|_| io::Error::other("closed"))?;
            }
            Ok(buf.len())
        }
        fn flush(&mut self) -> io::Result<()> {
            Ok(())
        }
    }

    #[test]
    fn follows_appends_and_rotation() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("app.log");
        fs::write(&path, "one\n").unwrap();
        let (tx, rx) = mpsc::channel();
        let p = path.clone();
        std::thread::spawn(move || {
            read(
                || Ok(p.clone()),
                &request(None, 1024, true),
                &mut Lines(tx, vec![]),
            )
        });
        let next = || rx.recv_timeout(Duration::from_secs(5)).unwrap();

        assert_eq!(next().data, "one\n");
        fs::OpenOptions::new()
            .append(true)
            .open(&path)
            .unwrap()
            .write_all(b"two\n")
            .unwrap();
        let two = next();
        assert_eq!(
            (two.offset, two.data.as_str(), two.rotated),
            (4, "two\n", false)
        );
        fs::write(&path, "new\n").unwrap(); // truncated
        let new = next();
        assert_eq!(
            (new.offset, new.data.as_str(), new.rotated),
            (0, "new\n", true)
        );
    }
}
