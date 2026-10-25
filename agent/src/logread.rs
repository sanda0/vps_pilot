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
