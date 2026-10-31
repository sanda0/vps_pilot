//! Dumps a project's database in the backup child, as the config file's
//! owner, with the tool installed on the server. Passwords never go on a
//! command line. See "Backing up" in docs/PROTOCOL.md.

use std::path::{Path, PathBuf};

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
            (server.username.as_str(), server.password.as_str()),
            ("shop", "a\"b\\c")
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
}
