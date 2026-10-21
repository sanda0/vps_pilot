//! Credentials persisted after enrollment.

use std::fs;
use std::io::Write;
use std::path::Path;

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct State {
    pub agent_id: String,
    pub node_key: String,
    pub node_id: i64,
    /// Pinned SHA-256 fingerprint of the hub's TLS certificate.
    pub hub_fingerprint: String,
}

impl State {
    /// The `connected` token: "<agent_id>:<node_key>".
    pub fn connect_token(&self) -> String {
        format!("{}:{}", self.agent_id, self.node_key)
    }

    /// Loads state; Ok(None) when the agent has not enrolled yet.
    pub fn load(path: &Path) -> Result<Option<State>> {
        match fs::read(path) {
            Ok(bytes) => {
                Ok(Some(serde_json::from_slice(&bytes).with_context(|| {
                    format!("parse state file {}", path.display())
                })?))
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
            Err(e) => Err(e).with_context(|| format!("read state file {}", path.display())),
        }
    }

    /// Writes state atomically (temp file + rename), readable only by the owner.
    pub fn save(&self, path: &Path) -> Result<()> {
        if let Some(dir) = path.parent().filter(|d| !d.as_os_str().is_empty()) {
            create_private_dir(dir)?;
        }
        let tmp = path.with_extension("json.tmp");
        {
            let mut file = open_private(&tmp)?;
            file.write_all(&serde_json::to_vec_pretty(self)?)?;
            file.sync_all()?;
        }
        fs::rename(&tmp, path).with_context(|| format!("write state file {}", path.display()))
    }
}

#[cfg(unix)]
fn open_private(path: &Path) -> Result<fs::File> {
    use std::os::unix::fs::OpenOptionsExt;
    Ok(fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .mode(0o600)
        .open(path)?)
}

#[cfg(not(unix))]
fn open_private(path: &Path) -> Result<fs::File> {
    Ok(fs::File::create(path)?)
}

#[cfg(unix)]
fn create_private_dir(dir: &Path) -> Result<()> {
    use std::os::unix::fs::DirBuilderExt;
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(dir)
        .with_context(|| format!("create state dir {}", dir.display()))
}

#[cfg(not(unix))]
fn create_private_dir(dir: &Path) -> Result<()> {
    Ok(fs::create_dir_all(dir)?)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample() -> State {
        State {
            agent_id: "id".into(),
            node_key: "key".into(),
            node_id: 4,
            hub_fingerprint: "ab".repeat(32),
        }
    }

    #[test]
    fn missing_file_is_not_enrolled() {
        let dir = tempfile::tempdir().unwrap();
        assert_eq!(State::load(&dir.path().join("state.json")).unwrap(), None);
    }

    #[test]
    fn save_then_load() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("nested/state.json");
        sample().save(&path).unwrap();
        assert_eq!(State::load(&path).unwrap(), Some(sample()));
        assert_eq!(sample().connect_token(), "id:key");

        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mode = fs::metadata(&path).unwrap().permissions().mode();
            assert_eq!(mode & 0o777, 0o600);
        }
    }
}
