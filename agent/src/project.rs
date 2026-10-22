//! A project's `config.vpspilot.json`, in the shape the hub expects
//! (`ProjectConfig` in server/internal/tcpserver/types.go).

use std::collections::HashSet;
use std::fs;
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
}

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
        let mut config: ProjectConfig =
            serde_json::from_slice(&bytes).with_context(|| format!("parse {}", file.display()))?;
        config
            .validate()
            .with_context(|| format!("invalid {}", file.display()))?;

        let dir = file.parent().unwrap_or(Path::new("."));
        let dir = fs::canonicalize(dir).with_context(|| format!("resolve {}", dir.display()))?;
        config.path = dir.to_string_lossy().into_owned();
        Ok(config)
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
    fn rejects_oversized_file() {
        let big = format!(r#"{{"name": "app", "tech": ["{}"]}}"#, "x".repeat(70_000));
        assert!(load_str(&big).is_err());
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
