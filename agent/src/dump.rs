//! Dumps a project's database in the backup child, as the config file's
//! owner, with the tool installed on the server. Passwords never go on a
//! command line. See "Backing up" in docs/PROTOCOL.md.

use std::fs::{self, File};
use std::io::{ErrorKind, Write};
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};

use crate::backupzip;
use crate::envfile::EnvFile;
use crate::project::ProjectBackupDatabase;

#[derive(Debug, PartialEq)]
pub enum Database {
    Mysql(Server),
    Postgres(Server),
    /// The resolved database file.
    Sqlite(PathBuf),
}

#[derive(Debug, PartialEq)]
pub struct Server {
    host: String,
    port: String,
    username: String,
    password: String,
    name: String,
}

impl Database {
    /// Reads the settings `names` points at from the env file.
    pub fn from_env(
        names: &ProjectBackupDatabase,
        env: &EnvFile,
        project: &Path,
    ) -> Result<Self, String> {
        // A name the config leaves empty means the setting isn't used.
        let var = |name: &str| {
            if name.is_empty() {
                Ok("")
            } else {
                env.get(name)
            }
        };
        let connection = var(&names.connection)?;
        let name = var(&names.database_name)?;
        if name.is_empty() {
            return Err("the database name is empty".into());
        }
        let server = || -> Result<Server, String> {
            if name.starts_with('-') {
                return Err(format!("invalid database name {name:?}"));
            }
            Ok(Server {
                host: var(&names.host)?.into(),
                port: var(&names.port)?.into(),
                username: var(&names.username)?.into(),
                password: var(&names.password)?.into(),
                name: name.into(),
            })
        };
        match connection {
            "mysql" | "mariadb" => Ok(Database::Mysql(server()?)),
            "pgsql" | "postgres" | "postgresql" => Ok(Database::Postgres(server()?)),
            "sqlite" => Ok(Database::Sqlite(backupzip::inside(project, name)?)),
            other => Err(format!("unsupported database connection {other:?}")),
        }
    }

    /// The `type` reported in BackupResult.database.
    pub fn kind(&self) -> &'static str {
        match self {
            Database::Mysql(_) => "mysql",
            Database::Postgres(_) => "postgres",
            Database::Sqlite(_) => "sqlite",
        }
    }

    /// The dump's name under `database/` in the zip.
    pub fn file_name(&self) -> &'static str {
        match self {
            Database::Sqlite(_) => "database.sqlite",
            _ => "dump.sql",
        }
    }

    /// Dumps into `tmp`, a private folder, and returns the dump file.
    pub fn dump(&self, tmp: &Path) -> Result<PathBuf, String> {
        let out = tmp.join(self.file_name());
        let mut cmd;
        match self {
            Database::Mysql(s) => {
                let defaults = tmp.join("my.cnf");
                write_private(&defaults, &mysql_defaults(s))?;
                cmd = Command::new("mysqldump");
                cmd.arg(format!("--defaults-extra-file={}", defaults.display()))
                    .args([
                        "--single-transaction",
                        "--quick",
                        "--routines",
                        "--triggers",
                    ])
                    .arg(format!("--result-file={}", out.display()))
                    .arg(&s.name);
            }
            Database::Postgres(s) => {
                cmd = Command::new("pg_dump");
                cmd.args(["--no-owner", "--format=plain"])
                    .arg(format!("--file={}", out.display()))
                    .arg(format!("--dbname={}", s.name))
                    .env("PGPASSWORD", &s.password);
                for (flag, value) in [
                    ("--host", &s.host),
                    ("--port", &s.port),
                    ("--username", &s.username),
                ] {
                    if !value.is_empty() {
                        cmd.arg(format!("{flag}={value}"));
                    }
                }
            }
            Database::Sqlite(file) => {
                // A relative target, so the path needs no quoting.
                cmd = Command::new("sqlite3");
                cmd.arg(file)
                    .arg(format!(".backup {}", self.file_name()))
                    .current_dir(tmp);
            }
        }
        run(cmd)?;
        Ok(out)
    }
}

/// Runs a dump tool. A failure says which tool and the last error line it
/// wrote to stderr.
fn run(mut cmd: Command) -> Result<(), String> {
    let tool = cmd.get_program().to_string_lossy().into_owned();
    let output = cmd
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::piped())
        .output()
        .map_err(|e| match e.kind() {
            ErrorKind::NotFound => format!("{tool} not found; install it on the server"),
            _ => format!("start {tool}: {e}"),
        })?;
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        // The last line that isn't an indented hint, like pg_dump's
        // "\tIs the server running on that host…".
        let last = stderr
            .lines()
            .rfind(|l| !l.trim().is_empty() && !l.starts_with(char::is_whitespace))
            .unwrap_or("");
        return Err(format!("{tool} failed ({}): {last}", output.status));
    }
    Ok(())
}

/// A MySQL option file with the connection settings, so the password isn't
/// on mysqldump's command line.
fn mysql_defaults(s: &Server) -> String {
    let quote = |v: &str| format!("\"{}\"", v.replace('\\', "\\\\").replace('"', "\\\""));
    let mut out = String::from("[client]\n");
    for (key, value) in [
        ("host", &s.host),
        ("port", &s.port),
        ("user", &s.username),
        ("password", &s.password),
    ] {
        if !value.is_empty() {
            out.push_str(&format!("{key}={}\n", quote(value)));
        }
    }
    out
}

fn write_private(path: &Path, text: &str) -> Result<(), String> {
    let _ = fs::remove_file(path);
    File::options()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .and_then(|mut f| f.write_all(text.as_bytes()))
        .map_err(|e| format!("write {}: {e}", path.display()))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn names() -> ProjectBackupDatabase {
        ProjectBackupDatabase {
            connection: "DB_CONNECTION".into(),
            host: "DB_HOST".into(),
            port: "DB_PORT".into(),
            username: "DB_USERNAME".into(),
            password: "DB_PASSWORD".into(),
            database_name: "DB_DATABASE".into(),
        }
    }

    #[test]
    fn reads_settings_from_the_env_file() {
        let project = Path::new("/srv/shop");
        let env = |text: &str| EnvFile::parse(text, ".env");
        let mysql = env(
            "DB_CONNECTION=mysql\nDB_HOST=127.0.0.1\nDB_PORT=3306\nDB_USERNAME=shop\nDB_PASSWORD=a\"b\\c\nDB_DATABASE=shop\n",
        );
        let db = Database::from_env(&names(), &mysql, project).unwrap();
        assert_eq!(db.kind(), "mysql");
        let Database::Mysql(server) = &db else {
            unreachable!()
        };
        assert_eq!(
            mysql_defaults(server),
            "[client]\nhost=\"127.0.0.1\"\nport=\"3306\"\nuser=\"shop\"\npassword=\"a\\\"b\\\\c\"\n"
        );

        let pg = env(
            "DB_CONNECTION=pgsql\nDB_HOST=db\nDB_PORT=\nDB_USERNAME=u\nDB_PASSWORD=\nDB_DATABASE=shop\n",
        );
        assert_eq!(
            Database::from_env(&names(), &pg, project).unwrap().kind(),
            "postgres"
        );

        for (text, want) in [
            (
                "DB_CONNECTION=mysql\nDB_DATABASE=shop\n",
                "DB_HOST is not set in .env",
            ),
            (
                "DB_CONNECTION=mongodb\nDB_DATABASE=shop\n",
                "unsupported database connection \"mongodb\"",
            ),
            (
                "DB_CONNECTION=mysql\nDB_DATABASE=\n",
                "the database name is empty",
            ),
        ] {
            assert_eq!(
                Database::from_env(&names(), &env(text), project).unwrap_err(),
                want
            );
        }
        let sqlite = env("DB_CONNECTION=sqlite\nDB_DATABASE=/etc/passwd\n");
        assert!(
            Database::from_env(&names(), &sqlite, project)
                .unwrap_err()
                .contains("outside")
        );
    }

    #[test]
    fn reports_missing_and_failing_tools() {
        assert_eq!(
            run(Command::new("vps-pilot-no-such-dump")).unwrap_err(),
            "vps-pilot-no-such-dump not found; install it on the server"
        );
        let mut sh = Command::new("sh");
        sh.args([
            "-c",
            "echo warming up >&2; echo access denied >&2; printf '\\thint\\n\\n' >&2; exit 2",
        ]);
        assert_eq!(
            run(sh).unwrap_err(),
            "sh failed (exit status: 2): access denied"
        );
    }
}
