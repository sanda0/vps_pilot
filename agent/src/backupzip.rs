//! Writes a backup zip in the backup child, as the config file's owner: the
//! declared folders and env file under `files/`, the database dump under
//! `database/` and a `manifest.json`. See "Backing up" in docs/PROTOCOL.md.

use std::collections::HashSet;
use std::fs::{self, File};
use std::io::{self, Write};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};

use zip::write::SimpleFileOptions;
use zip::{CompressionMethod, DateTime, ZipWriter};

/// What goes into a zip. `roots` are files or folders inside `project`.
pub struct Contents<'a> {
    /// The resolved project directory.
    pub project: &'a Path,
    pub roots: Vec<PathBuf>,
    /// The dump file and its name under `database/`.
    pub database: Option<(PathBuf, &'a str)>,
    pub manifest: serde_json::Value,
}

/// Resolves `path` (`..` and symlinks), relative to the resolved project
/// directory or absolute, and checks that it stays inside it.
pub fn inside(project: &Path, path: &str) -> Result<PathBuf, String> {
    let real = fs::canonicalize(project.join(path)).map_err(|e| format!("{path}: {e}"))?;
    if !real.starts_with(project) {
        return Err(format!("{path} resolves outside the project directory"));
    }
    Ok(real)
}

/// Writes the zip to `part`, which must not exist, and returns its size.
pub fn write(part: &Path, contents: &Contents) -> Result<u64, String> {
    let file = File::options()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(part)
        .map_err(|e| format!("create {}: {e}", part.display()))?;
    let mut zip = ZipWriter::new(file);
    let failed = |e: &dyn std::fmt::Display| format!("write the zip: {e}");

    let manifest = serde_json::to_vec_pretty(&contents.manifest).map_err(|e| failed(&e))?;
    zip.start_file("manifest.json", options(None))
        .map_err(|e| failed(&e))?;
    zip.write_all(&manifest).map_err(|e| failed(&e))?;

    let mut seen = HashSet::new();
    for root in &contents.roots {
        add(&mut zip, contents.project, root, &mut seen)?;
    }
    if let Some((dump, name)) = &contents.database {
        add_file(&mut zip, dump, &format!("database/{name}"))?;
    }
    let file = zip.finish().map_err(|e| failed(&e))?;
    file.sync_all().map_err(|e| failed(&e))?;
    Ok(file.metadata().map_err(|e| failed(&e))?.len())
}

/// Adds a file, or a folder's files, under `files/<path in the project>`.
/// Symlinks are not followed or stored.
fn add(
    zip: &mut ZipWriter<File>,
    project: &Path,
    path: &Path,
    seen: &mut HashSet<PathBuf>,
) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|e| format!("{}: {e}", path.display()))?;
    if meta.is_dir() {
        let mut names: Vec<_> = fs::read_dir(path)
            .and_then(|d| d.map(|e| e.map(|e| e.path())).collect::<io::Result<_>>())
            .map_err(|e| format!("{}: {e}", path.display()))?;
        names.sort();
        for name in names {
            add(zip, project, &name, seen)?;
        }
    } else if meta.is_file() && seen.insert(path.to_path_buf()) {
        let rel = path.strip_prefix(project).unwrap_or(path);
        add_file(zip, path, &format!("files/{}", rel.to_string_lossy()))?;
    }
    Ok(())
}

fn add_file(zip: &mut ZipWriter<File>, path: &Path, name: &str) -> Result<(), String> {
    let failed = |e: &dyn std::fmt::Display| format!("{}: {e}", path.display());
    let mut file = File::open(path).map_err(|e| failed(&e))?;
    let meta = file.metadata().map_err(|e| failed(&e))?;
    let opts = options(Some(&meta)).unix_permissions(meta.mode() & 0o777);
    zip.start_file(name, opts).map_err(|e| failed(&e))?;
    io::copy(&mut file, zip).map_err(|e| failed(&e))?;
    Ok(())
}

fn options(meta: Option<&fs::Metadata>) -> SimpleFileOptions {
    let mut opts = SimpleFileOptions::default().compression_method(CompressionMethod::Deflated);
    if let Some(meta) = meta {
        opts = opts.large_file(meta.len() >= u32::MAX as u64);
        let (y, mo, d, h, mi, s) = civil_time(meta.mtime().max(0) as u64);
        if let Ok(t) = DateTime::from_date_and_time(y as u16, mo, d, h, mi, s) {
            opts = opts.last_modified_time(t);
        }
    }
    opts
}

/// Unix seconds as `20261031T030000Z`, the time in a zip's name.
pub fn stamp(secs: u64) -> String {
    let (y, mo, d, h, mi, s) = civil_time(secs);
    format!("{y:04}{mo:02}{d:02}T{h:02}{mi:02}{s:02}Z")
}

/// The reverse of `stamp`.
pub fn parse_stamp(s: &str) -> Option<u64> {
    let b = s.as_bytes();
    if b.len() != 16 || b[8] != b'T' || b[15] != b'Z' {
        return None;
    }
    let num = |r: std::ops::Range<usize>| s.get(r)?.parse::<u32>().ok();
    let (y, mo, d) = (num(0..4)?, num(4..6)?, num(6..8)?);
    let (h, mi, sec) = (num(9..11)?, num(11..13)?, num(13..15)?);
    if !(1..=12).contains(&mo) || !(1..=31).contains(&d) || h > 23 || mi > 59 || sec > 59 {
        return None;
    }
    let days = days_from_civil(y as i64, mo as i64, d as i64);
    u64::try_from(days * 86400 + (h * 3600 + mi * 60 + sec) as i64).ok()
}

/// UTC (year, month, day, hour, minute, second) of Unix seconds.
fn civil_time(secs: u64) -> (u32, u8, u8, u8, u8, u8) {
    let (days, rem) = ((secs / 86400) as i64, secs % 86400);
    // Howard Hinnant's civil_from_days.
    let z = days + 719_468;
    let (era, doe) = (z.div_euclid(146_097), z.rem_euclid(146_097));
    let yoe = (doe - doe / 1460 + doe / 36_524 - doe / 146_096) / 365;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = yoe + era * 400 + i64::from(m <= 2);
    let (h, mi, s) = (rem / 3600, rem % 3600 / 60, rem % 60);
    (y as u32, m as u8, d as u8, h as u8, mi as u8, s as u8)
}

fn days_from_civil(y: i64, m: i64, d: i64) -> i64 {
    let y = if m <= 2 { y - 1 } else { y };
    let era = y.div_euclid(400);
    let yoe = y - era * 400;
    let doy = (153 * ((m + 9) % 12) + 2) / 5 + d - 1;
    era * 146_097 + yoe * 365 + yoe / 4 - yoe / 100 + doy - 719_468
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Read;

    #[test]
    fn stamps_round_trip() {
        for (secs, want) in [
            (0, "19700101T000000Z"),
            (951_782_400, "20000229T000000Z"),
            (1_793_415_600, "20261031T030000Z"),
        ] {
            assert_eq!(stamp(secs), want);
            assert_eq!(parse_stamp(want), Some(secs));
        }
        for bad in ["20261031T030000", "20261331T030000Z", "2026103xT030000Z"] {
            assert_eq!(parse_stamp(bad), None, "{bad}");
        }
    }

    #[test]
    fn zips_files_skips_symlinks_and_stays_inside() {
        let tmp = tempfile::tempdir().unwrap();
        let project = fs::canonicalize(tmp.path()).unwrap();
        fs::create_dir_all(project.join("storage/app/sub")).unwrap();
        fs::write(project.join("storage/app/sub/a.txt"), "a").unwrap();
        fs::write(project.join(".env"), "DB_PASSWORD=x\n").unwrap();
        fs::write(project.join("dump.sql"), "-- dump").unwrap();
        std::os::unix::fs::symlink("/etc", project.join("storage/app/etc")).unwrap();
        assert!(inside(&project, "../").unwrap_err().contains("outside"));
        assert!(
            inside(&project, "storage/app/etc")
                .unwrap_err()
                .contains("outside")
        );

        let part = project.join("b.zip.part");
        let contents = Contents {
            project: &project,
            roots: vec![
                inside(&project, "storage/app").unwrap(),
                inside(&project, ".env").unwrap(),
                inside(&project, "storage/app/sub/a.txt").unwrap(),
            ],
            database: Some((project.join("dump.sql"), "dump.sql")),
            manifest: serde_json::json!({"project": "shop"}),
        };
        let size = write(&part, &contents).unwrap();
        assert_eq!(size, fs::metadata(&part).unwrap().len());
        assert_eq!(fs::metadata(&part).unwrap().mode() & 0o777, 0o600);
        assert!(
            write(&part, &contents).is_err(),
            "overwrote an existing file"
        );

        let mut zip = zip::ZipArchive::new(File::open(&part).unwrap()).unwrap();
        let mut names: Vec<_> = zip.file_names().map(|n| n.unwrap().into_owned()).collect();
        names.sort();
        assert_eq!(
            names,
            [
                "database/dump.sql",
                "files/.env",
                "files/storage/app/sub/a.txt",
                "manifest.json"
            ]
        );
        let mut env = String::new();
        zip.by_name("files/.env")
            .unwrap()
            .read_to_string(&mut env)
            .unwrap();
        assert_eq!(env, "DB_PASSWORD=x\n");
    }
}
