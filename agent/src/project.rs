//! A project's `config.vpspilot.json`, in the shape the hub expects
//! (`ProjectConfig` in server/internal/tcpserver/types.go).

use std::collections::HashSet;
use std::fs::{self, File};
use std::io::{ErrorKind, Read};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::Path;

use anyhow::{Context, Result, bail};
use serde::{Deserialize, Serialize};

pub const CONFIG_FILE: &str = "config.vpspilot.json";

/// Config files larger than this are rejected.
const MAX_CONFIG_SIZE: u64 = 64 * 1024;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ProjectConfig {
    pub name: String,
    /// Absolute path of the project directory. Set by the agent, not read
    /// from the file.
    #[serde(default, skip_deserializing)]
    pub path: String,
    #[serde(default)]
    pub tech: Vec<String>,
    /// The only commands the hub can ask this agent to run, by name.
    #[serde(default)]
    pub commands: Vec<ProjectCommand>,
    #[serde(default)]
    pub logs: Vec<String>,
    #[serde(default)]
    pub backups: Option<ProjectBackup>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ProjectCommand {
    pub name: String,
    pub command: String,
}

#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct ProjectBackup {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub env_file: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub zip_file_name: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub database: Option<ProjectBackupDatabase>,
    #[serde(default, rename = "dir", skip_serializing_if = "Vec::is_empty")]
    pub dirs: Vec<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub schedule: Option<BackupSchedule>,
    /// `HH:MM` in the server's local time zone; see `time_of_day`.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub at: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub local_dir: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub retention_days: Option<u32>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub s3: Option<ProjectBackupS3>,
}

#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum BackupSchedule {
    Daily,
    /// Sundays.
    Weekly,
}

/// Where to upload backups. The keys are the *names* of env-file variables,
/// like `ProjectBackupDatabase`, so the hub never sees them.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct ProjectBackupS3 {
    pub access_key: String,
    pub secret_key: String,
    pub bucket: String,
    pub region: String,
    pub endpoint: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub prefix: String,
    pub path_style: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub retention_days: Option<u32>,
}

const DEFAULT_RETENTION_DAYS: u32 = 7;
const DEFAULT_S3_RETENTION_DAYS: u32 = 30;
const MAX_RETENTION_DAYS: u32 = 3650;

/// Names of the env-file variables that hold the database settings.
#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct ProjectBackupDatabase {
    pub connection: String,
    pub host: String,
    pub port: String,
    pub username: String,
    pub password: String,
    pub database_name: String,
}

impl ProjectBackup {
    /// When a scheduled backup runs, as (hour, minute); 03:00 by default.
    pub fn time_of_day(&self) -> (u32, u32) {
        parse_time(&self.at).unwrap_or((3, 0))
    }

    pub fn retention_days(&self) -> u32 {
        self.retention_days.unwrap_or(DEFAULT_RETENTION_DAYS)
    }

    /// Base name of the zip: `zip_file_name`, or the project's name with
    /// anything but letters, digits, `.`, `-` and `_` replaced by `_`.
    /// `zip_file_name` predates backups running, so a bad one only fails
    /// the backup instead of making the whole config unusable.
    pub fn zip_base(&self, project_name: &str) -> Result<String> {
        if self.zip_file_name.is_empty() {
            return Ok(project_name
                .chars()
                .map(|c| if zip_char(c) { c } else { '_' })
                .collect());
        }
        if !self.zip_file_name.chars().all(zip_char) || self.zip_file_name.starts_with('.') {
            bail!("zip_file_name may only hold letters, digits, \".\", \"-\" and \"_\"");
        }
        Ok(self.zip_file_name.clone())
    }

    /// Checks the fields added with backups in v0.5. Configs written before
    /// can't hold them, so rejecting the config doesn't break anyone.
    fn validate(&self) -> Result<()> {
        if !self.at.is_empty() && parse_time(&self.at).is_none() {
            bail!("backups.at must be HH:MM, e.g. \"03:00\"");
        }
        check_retention("backups.retention_days", self.retention_days)?;
        if let Some(s3) = &self.s3 {
            for (field, value) in [
                ("access_key", &s3.access_key),
                ("secret_key", &s3.secret_key),
                ("bucket", &s3.bucket),
                ("region", &s3.region),
                ("endpoint", &s3.endpoint),
            ] {
                if value.trim().is_empty() {
                    bail!("backups.s3.{field} is required");
                }
            }
            if !s3.endpoint.starts_with("https://") {
                bail!("backups.s3.endpoint must start with https://");
            }
            if self.env_file.is_empty() {
                bail!("backups.s3 needs backups.env_file, which holds the keys");
            }
            check_retention("backups.s3.retention_days", s3.retention_days)?;
        }
        Ok(())
    }
}

impl ProjectBackupS3 {
    pub fn retention_days(&self) -> u32 {
        self.retention_days.unwrap_or(DEFAULT_S3_RETENTION_DAYS)
    }
}

#[allow(dead_code)]
fn zip_char(c: char) -> bool {
    c.is_ascii_alphanumeric() || matches!(c, '.' | '-' | '_')
}

fn parse_time(s: &str) -> Option<(u32, u32)> {
    let (h, m) = s.split_once(':')?;
    if h.len() != 2 || m.len() != 2 {
        return None;
    }
    let (h, m) = (h.parse().ok()?, m.parse().ok()?);
    (h < 24 && m < 60).then_some((h, m))
}

fn check_retention(field: &str, days: Option<u32>) -> Result<()> {
    match days {
        Some(d) if !(1..=MAX_RETENTION_DAYS).contains(&d) => {
            bail!("{field} must be between 1 and {MAX_RETENTION_DAYS}")
        }
        _ => Ok(()),
    }
}

impl ProjectConfig {
    /// Reads and validates a config file. `path` is set to the absolute
    /// directory that contains it.
    pub fn load(file: &Path) -> Result<Self> {
        let size = fs::metadata(file)
            .with_context(|| format!("read {}", file.display()))?
            .len();
        if size > MAX_CONFIG_SIZE {
            bail!("{} is too large ({size} bytes)", file.display());
        }
        let bytes = fs::read(file).with_context(|| format!("read {}", file.display()))?;
        Self::parse(&bytes, file)
    }

    /// Re-reads `<dir>/config.vpspilot.json` just before running one of its
    /// commands, and returns it with the uid of the file's owner. Stricter
    /// than `load`, because the owner is who the commands run as: the file
    /// must not be a symlink or writable by group or others. It is opened
    /// with O_NOFOLLOW and checked with fstat, so it can't be swapped between
    /// the check and the read.
    pub fn load_for_run(dir: &Path) -> Result<(Self, u32)> {
        let file = dir.join(CONFIG_FILE);
        let mut f = match File::options()
            .read(true)
            .custom_flags(nix::fcntl::OFlag::O_NOFOLLOW.bits())
            .open(&file)
        {
            Ok(f) => f,
            Err(e) if e.raw_os_error() == Some(nix::libc::ELOOP) => {
                bail!("{CONFIG_FILE} is a symlink")
            }
            Err(e) if e.kind() == ErrorKind::NotFound => bail!("{CONFIG_FILE} not found"),
            Err(e) => return Err(e).with_context(|| format!("read {}", file.display())),
        };
        let meta = f.metadata()?;
        if !meta.is_file() {
            bail!("{CONFIG_FILE} is not a regular file");
        }
        if meta.mode() & 0o022 != 0 {
            bail!("{CONFIG_FILE} is group- or world-writable; run chmod 644 {CONFIG_FILE}");
        }
        if meta.len() > MAX_CONFIG_SIZE {
            bail!("{CONFIG_FILE} is too large ({} bytes)", meta.len());
        }
        let mut bytes = Vec::new();
        f.by_ref()
            .take(MAX_CONFIG_SIZE + 1)
            .read_to_end(&mut bytes)?;
        let config = Self::parse(&bytes, &file)?;
        Ok((config, meta.uid()))
    }

    fn parse(bytes: &[u8], file: &Path) -> Result<Self> {
        if bytes.len() as u64 > MAX_CONFIG_SIZE {
            bail!("{} is too large ({} bytes)", file.display(), bytes.len());
        }
        let mut config: ProjectConfig =
            serde_json::from_slice(bytes).with_context(|| format!("parse {}", file.display()))?;
        config
            .validate()
            .with_context(|| format!("invalid {}", file.display()))?;

        let dir = file.parent().unwrap_or(Path::new("."));
        let dir = fs::canonicalize(dir).with_context(|| format!("resolve {}", dir.display()))?;
        config.path = dir.to_string_lossy().into_owned();
        Ok(config)
    }

    /// The command string for a command name.
    pub fn command(&self, name: &str) -> Option<&str> {
        self.commands
            .iter()
            .find(|c| c.name == name)
            .map(|c| c.command.as_str())
    }

    fn validate(&mut self) -> Result<()> {
        self.name = self.name.trim().to_string();
        if self.name.is_empty() {
            bail!("\"name\" is required");
        }
        let mut seen = HashSet::new();
        for cmd in &self.commands {
            if cmd.name.trim().is_empty() || cmd.command.trim().is_empty() {
                bail!("every command needs a \"name\" and a \"command\"");
            }
            if !seen.insert(cmd.name.as_str()) {
                bail!("command name {:?} is used twice", cmd.name);
            }
        }
        if let Some(backups) = &self.backups {
            backups.validate()?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn load_str(json: &str) -> Result<ProjectConfig> {
        let dir = tempfile::tempdir().unwrap();
        let file = dir.path().join(CONFIG_FILE);
        fs::write(&file, json).unwrap();
        ProjectConfig::load(&file)
    }

    #[test]
    fn loads_full_example() {
        let config = load_str(
            r#"{
              "name": "meta ads dashboard",
              "tech": ["laravel", "react", "mysql"],
              "logs": ["storage/logs"],
              "commands": [
                { "name": "node build", "command": "npm run build" },
                { "name": "php build", "command": "composer install" }
              ],
              "backups": {
                "env_file": ".env",
                "zip_file_name": "project_backup",
                "database": { "connection": "DB_CONNECTION", "host": "DB_HOST" },
                "dir": ["storage/app"]
              }
            }"#,
        )
        .unwrap();
        assert_eq!(config.name, "meta ads dashboard");
        assert_eq!(config.commands[1].command, "composer install");
        let backups = config.backups.unwrap();
        assert_eq!(backups.dirs, ["storage/app"]);
        assert_eq!(backups.database.unwrap().host, "DB_HOST");
    }

    #[test]
    fn minimal_config_gets_project_dir_as_path() {
        let dir = tempfile::tempdir().unwrap();
        let file = dir.path().join(CONFIG_FILE);
        fs::write(&file, r#"{"name": "  app  ", "path": "/ignored"}"#).unwrap();
        let config = ProjectConfig::load(&file).unwrap();
        assert_eq!(config.name, "app");
        assert!(config.commands.is_empty() && config.backups.is_none());
        let want = fs::canonicalize(dir.path()).unwrap();
        assert_eq!(Path::new(&config.path), want);
    }

    #[test]
    fn rejects_invalid_configs() {
        for bad in [
            "not json",
            r#"{"tech": []}"#,
            r#"{"name": " "}"#,
            r#"{"name": "app", "commands": [{"name": "build", "command": ""}]}"#,
            r#"{"name": "app", "commands": [
                {"name": "build", "command": "make"},
                {"name": "build", "command": "make all"}]}"#,
        ] {
            assert!(load_str(bad).is_err(), "{bad}");
        }
    }

    #[test]
    fn loads_backup_schedule_and_s3() {
        let config = load_str(
            r#"{"name": "shop", "backups": {
                "env_file": ".env", "dir": ["storage/app"], "local_dir": "../backups/shop",
                "schedule": "weekly", "at": "23:45", "retention_days": 14,
                "s3": {"access_key": "S3_KEY", "secret_key": "S3_SECRET", "bucket": "b",
                       "region": "us-east-1", "endpoint": "https://us-east-1.linodeobjects.com"}}}"#,
        )
        .unwrap();
        let backups = config.backups.unwrap();
        assert_eq!(backups.schedule, Some(BackupSchedule::Weekly));
        assert_eq!(backups.time_of_day(), (23, 45));
        assert_eq!(backups.retention_days(), 14);
        let s3 = backups.s3.as_ref().unwrap();
        assert_eq!(
            (s3.secret_key.as_str(), s3.retention_days()),
            ("S3_SECRET", 30)
        );
        assert!(!s3.path_style && s3.prefix.is_empty());
        assert_eq!(backups.zip_base("my shop").unwrap(), "my_shop");
    }

    #[test]
    fn rejects_invalid_backup_fields() {
        let s3 = r#""access_key": "K", "secret_key": "S", "bucket": "b", "region": "r""#;
        for bad in [
            r#"{"schedule": "hourly"}"#.to_string(),
            r#"{"schedule": "daily", "at": "3:00"}"#.into(),
            r#"{"at": "24:00"}"#.into(),
            r#"{"retention_days": 0}"#.into(),
            r#"{"retention_days": 4000}"#.into(),
            format!(r#"{{"env_file": ".env", "s3": {{{s3}}}}}"#),
            format!(r#"{{"env_file": ".env", "s3": {{{s3}, "endpoint": "http://minio:9000"}}}}"#),
            format!(r#"{{"s3": {{{s3}, "endpoint": "https://s3.example.com"}}}}"#),
            format!(
                r#"{{"env_file": ".env", "s3": {{{s3}, "endpoint": "https://e", "retention_days": 0}}}}"#
            ),
        ] {
            let json = format!(r#"{{"name": "app", "backups": {bad}}}"#);
            assert!(load_str(&json).is_err(), "{bad}");
        }
    }

    #[test]
    fn older_backup_fields_fail_only_the_backup() {
        // Accepted before backups ran: no local_dir, a zip name with a space.
        let config = load_str(
            r#"{"name": "app", "backups": {"zip_file_name": "app backup", "dir": ["data"]}}"#,
        )
        .unwrap();
        let backups = config.backups.unwrap();
        assert!(backups.zip_base("app").is_err());
        assert_eq!(backups.time_of_day(), (3, 0));
        assert_eq!(backups.retention_days(), 7);
    }

    #[test]
    fn rejects_oversized_file() {
        let big = format!(r#"{{"name": "app", "tech": ["{}"]}}"#, "x".repeat(70_000));
        assert!(load_str(&big).is_err());
    }

    #[test]
    fn load_for_run_returns_owner_and_commands() {
        use std::os::unix::fs::PermissionsExt;
        let dir = tempfile::tempdir().unwrap();
        let file = dir.path().join(CONFIG_FILE);
        fs::write(
            &file,
            r#"{"name": "app", "commands": [{"name": "build", "command": "make"}]}"#,
        )
        .unwrap();
        fs::set_permissions(&file, fs::Permissions::from_mode(0o644)).unwrap();
        let (config, uid) = ProjectConfig::load_for_run(dir.path()).unwrap();
        assert_eq!(config.command("build"), Some("make"));
        assert_eq!(config.command("deploy"), None);
        assert_eq!(uid, nix::unistd::getuid().as_raw());

        for mode in [0o664, 0o646] {
            fs::set_permissions(&file, fs::Permissions::from_mode(mode)).unwrap();
            let err = ProjectConfig::load_for_run(dir.path()).unwrap_err();
            assert!(err.to_string().contains("writable"), "{mode:o}: {err}");
        }
    }

    #[test]
    fn load_for_run_rejects_symlinked_config() {
        let real = tempfile::tempdir().unwrap();
        fs::write(real.path().join(CONFIG_FILE), r#"{"name": "app"}"#).unwrap();
        let dir = tempfile::tempdir().unwrap();
        std::os::unix::fs::symlink(real.path().join(CONFIG_FILE), dir.path().join(CONFIG_FILE))
            .unwrap();
        let err = ProjectConfig::load_for_run(dir.path()).unwrap_err();
        assert!(err.to_string().contains("symlink"), "{err}");
    }

    #[test]
    fn serializes_in_hub_shape() {
        let mut config = load_str(
            r#"{"name": "app", "commands": [{"name": "build", "command": "make"}],
                "backups": {"dir": ["data"]}}"#,
        )
        .unwrap();
        config.path = "/srv/app".into();
        assert_eq!(
            serde_json::to_value(&config).unwrap(),
            serde_json::json!({
                "name": "app",
                "path": "/srv/app",
                "tech": [],
                "commands": [{"name": "build", "command": "make"}],
                "logs": [],
                "backups": {"dir": ["data"]}
            })
        );
    }
}
