//! A project's env file (`backups.env_file`), read in the backup child for
//! the database and S3 settings the config names. See "Backing up" in
//! docs/PROTOCOL.md.

use std::collections::HashMap;
use std::fs::File;
use std::io::Read;
use std::path::Path;

/// Env files larger than this are rejected.
const MAX_SIZE: u64 = 1024 * 1024;

#[derive(Debug, Default)]
pub struct EnvFile {
    /// The file's name as the config gives it, for error messages.
    name: String,
    vars: HashMap<String, String>,
}

impl EnvFile {
    pub fn load(path: &Path, name: &str) -> Result<Self, String> {
        let mut text = String::new();
        File::open(path)
            .and_then(|f| f.take(MAX_SIZE + 1).read_to_string(&mut text))
            .map_err(|e| format!("read {name}: {e}"))?;
        if text.len() as u64 > MAX_SIZE {
            return Err(format!("{name} is larger than 1 MiB"));
        }
        Ok(Self::parse(&text, name))
    }

    /// `KEY=VALUE` lines. Blank lines, `#` comments and an `export ` prefix
    /// are ignored, and quotes around a value are removed. An unquoted value
    /// ends at ` #`. There is no `${…}` expansion.
    pub fn parse(text: &str, name: &str) -> Self {
        let mut vars = HashMap::new();
        for line in text.lines() {
            let line = line.trim();
            let line = line.strip_prefix("export ").unwrap_or(line);
            if line.starts_with('#') {
                continue;
            }
            let Some((key, value)) = line.split_once('=') else {
                continue;
            };
            vars.insert(key.trim().to_string(), unquote(value.trim()).to_string());
        }
        EnvFile {
            name: name.to_string(),
            vars,
        }
    }

    /// The value of the variable called `var`, which must be set.
    pub fn get(&self, var: &str) -> Result<&str, String> {
        self.vars
            .get(var)
            .map(String::as_str)
            .ok_or_else(|| format!("{var} is not set in {}", self.name))
    }
}

fn unquote(value: &str) -> &str {
    for q in ['"', '\''] {
        if let Some(inner) = value.strip_prefix(q)
            && let Some(end) = inner.find(q)
        {
            return &inner[..end];
        }
    }
    match value.find(" #") {
        Some(i) => value[..i].trim_end(),
        None => value,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_dotenv_lines() {
        let env = EnvFile::parse(
            r#"
# Database
APP_NAME="My Shop"
DB_HOST=127.0.0.1
export DB_PORT = 3306
DB_PASSWORD='p#ss "word"'
DB_USERNAME=shop # the app user
EMPTY=
URL=https://example.com/?a=b
not a variable
"#,
            ".env",
        );
        for (var, want) in [
            ("APP_NAME", "My Shop"),
            ("DB_HOST", "127.0.0.1"),
            ("DB_PORT", "3306"),
            ("DB_PASSWORD", "p#ss \"word\""),
            ("DB_USERNAME", "shop"),
            ("EMPTY", ""),
            ("URL", "https://example.com/?a=b"),
        ] {
            assert_eq!(env.get(var), Ok(want), "{var}");
        }
        assert_eq!(
            env.get("DB_DATABASE"),
            Err("DB_DATABASE is not set in .env".into())
        );
    }

    #[test]
    fn rejects_large_files() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join(".env");
        std::fs::write(&path, "A=1\n".repeat(300_000)).unwrap();
        assert!(EnvFile::load(&path, ".env").unwrap_err().contains("1 MiB"));
        std::fs::write(&path, "A=1\n").unwrap();
        assert_eq!(EnvFile::load(&path, ".env").unwrap().get("A"), Ok("1"));
    }
}
