//! A small S3 client for the backup child: upload a zip, list and delete
//! old ones. Works with any S3-compatible storage (Linode Object Storage,
//! AWS, MinIO). See "Backing up" in docs/PROTOCOL.md.

use std::time::{Duration, SystemTime, UNIX_EPOCH};

use crate::backupzip;
use crate::sigv4::{self, Credentials};

const TIMEOUT: Duration = Duration::from_secs(10 * 60);

pub struct Bucket {
    agent: ureq::Agent,
    /// `https` (or `http` in tests).
    scheme: String,
    /// The endpoint's host, with a port if it has one.
    host: String,
    bucket: String,
    path_style: bool,
    region: String,
    access_key: String,
    secret_key: String,
}

struct Reply {
    etag: Option<String>,
    body: String,
}

impl Bucket {
    pub fn new(
        endpoint: &str,
        bucket: &str,
        region: &str,
        access_key: &str,
        secret_key: &str,
        path_style: bool,
    ) -> Result<Self, String> {
        let (scheme, rest) = endpoint
            .split_once("://")
            .ok_or_else(|| format!("invalid endpoint {endpoint:?}"))?;
        let host = rest.trim_end_matches('/');
        if host.is_empty() || host.contains('/') {
            return Err(format!("invalid endpoint {endpoint:?}"));
        }
        let agent = ureq::Agent::config_builder()
            .http_status_as_error(false)
            .timeout_global(Some(TIMEOUT))
            .build()
            .into();
        Ok(Bucket {
            agent,
            scheme: scheme.into(),
            host: host.into(),
            bucket: bucket.into(),
            path_style,
            region: region.into(),
            access_key: access_key.into(),
            secret_key: secret_key.into(),
        })
    }

    /// Sends a signed request for `key` ("" for the bucket) and returns the
    /// reply, or an error for a status other than 2xx.
    fn request(
        &self,
        method: &str,
        key: &str,
        query: &[(&str, &str)],
        body: &[u8],
    ) -> Result<Reply, String> {
        let (host, path) = self.location(key);
        let mut query: Vec<(String, String)> = query
            .iter()
            .map(|(k, v)| (sigv4::encode(k, false), sigv4::encode(v, false)))
            .collect();
        query.sort();
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_or(0, |d| d.as_secs());
        let amz_date = backupzip::stamp(now);
        let payload_hash = sigv4::sha256_hex(body);
        let headers = [
            ("host", host.clone()),
            ("x-amz-content-sha256", payload_hash.clone()),
            ("x-amz-date", amz_date.clone()),
        ];
        let creds = Credentials {
            access_key: &self.access_key,
            secret_key: &self.secret_key,
            region: &self.region,
        };
        let auth = sigv4::authorization(
            &creds,
            &sigv4::Request {
                method,
                path: &path,
                query: &query,
                headers: &headers,
                payload_hash: &payload_hash,
                amz_date: &amz_date,
            },
        );
        let query: Vec<String> = query.iter().map(|(k, v)| format!("{k}={v}")).collect();
        let mut url = format!("{}://{host}{path}", self.scheme);
        if !query.is_empty() {
            url = format!("{url}?{}", query.join("&"));
        }
        let mut req = ureq::http::Request::builder()
            .method(method)
            .uri(&url)
            .header("authorization", auth);
        for (name, value) in &headers[1..] {
            req = req.header(*name, value);
        }
        let req = req.body(body).map_err(|e| e.to_string())?;
        let failed = |e: &dyn std::fmt::Display| format!("{method} {}: {e}", self.describe(key));
        let mut resp = self.agent.run(req).map_err(|e| failed(&e))?;
        let status = resp.status();
        let etag = resp
            .headers()
            .get("etag")
            .and_then(|v| v.to_str().ok())
            .map(String::from);
        let body = resp.body_mut().read_to_string().unwrap_or_default();
        if !status.is_success() {
            let why = error_message(&body).unwrap_or_else(|| status.to_string());
            return Err(failed(&why));
        }
        Ok(Reply { etag, body })
    }

    /// The host and URI-encoded path for `key`.
    fn location(&self, key: &str) -> (String, String) {
        let key = sigv4::encode(key, true);
        if self.path_style {
            let path = format!("/{}", sigv4::encode(&self.bucket, false));
            let path = if key.is_empty() {
                path
            } else {
                format!("{path}/{key}")
            };
            (self.host.clone(), path)
        } else {
            (format!("{}.{}", self.bucket, self.host), format!("/{key}"))
        }
    }

    fn describe(&self, key: &str) -> String {
        format!("s3://{}/{key}", self.bucket)
    }
}

/// The text of every `<name>…</name>` in `xml`, unescaped.
fn tags(xml: &str, name: &str) -> Vec<String> {
    let (open, close) = (format!("<{name}>"), format!("</{name}>"));
    let mut out = Vec::new();
    let mut rest = xml;
    while let Some(start) = rest.find(&open) {
        rest = &rest[start + open.len()..];
        let Some(end) = rest.find(&close) else { break };
        out.push(unescape(&rest[..end]));
        rest = &rest[end + close.len()..];
    }
    out
}

fn unescape(s: &str) -> String {
    s.replace("&lt;", "<")
        .replace("&gt;", ">")
        .replace("&quot;", "\"")
        .replace("&apos;", "'")
        .replace("&#34;", "\"")
        .replace("&amp;", "&")
}

/// `Code: Message` from an S3 `<Error>` reply.
fn error_message(xml: &str) -> Option<String> {
    if !xml.contains("<Error>") {
        return None;
    }
    let code = tags(xml, "Code").pop().unwrap_or_default();
    match tags(xml, "Message").pop() {
        Some(msg) if !msg.is_empty() => Some(format!("{code}: {msg}")),
        _ => Some(code),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn locates_keys_both_styles() {
        let mut b = Bucket::new(
            "https://us-east-1.linodeobjects.com/",
            "shop",
            "us-east-1",
            "k",
            "s",
            false,
        )
        .unwrap();
        assert_eq!(
            b.location("daily/a b.zip"),
            (
                "shop.us-east-1.linodeobjects.com".into(),
                "/daily/a%20b.zip".into()
            )
        );
        b.path_style = true;
        assert_eq!(
            b.location("a.zip"),
            ("us-east-1.linodeobjects.com".into(), "/shop/a.zip".into())
        );
        assert_eq!(
            b.location(""),
            ("us-east-1.linodeobjects.com".into(), "/shop".into())
        );
        assert!(Bucket::new("https://host/path", "b", "r", "k", "s", false).is_err());
    }

    #[test]
    fn reads_s3_xml() {
        let err = "<?xml version=\"1.0\"?><Error><Code>AccessDenied</Code><Message>Access &amp; denied</Message></Error>";
        assert_eq!(error_message(err).unwrap(), "AccessDenied: Access & denied");
        assert_eq!(error_message("<InitiateMultipartUploadResult/>"), None);
        let list = "<ListBucketResult><Contents><Key>a.zip</Key></Contents><Contents><Key>b&amp;c.zip</Key></Contents></ListBucketResult>";
        assert_eq!(tags(list, "Key"), ["a.zip", "b&c.zip"]);
    }
}
