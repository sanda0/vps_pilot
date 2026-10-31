//! AWS Signature Version 4, for the S3 requests the backup child makes.
//! https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html

use hmac::{Hmac, KeyInit, Mac};
use sha2::{Digest, Sha256};

pub struct Credentials<'a> {
    pub access_key: &'a str,
    pub secret_key: &'a str,
    pub region: &'a str,
}

/// A request to sign. `path` and the query are already URI-encoded, with
/// the query sorted by name.
pub struct Request<'a> {
    pub method: &'a str,
    pub path: &'a str,
    pub query: &'a [(String, String)],
    /// Lowercase names, sorted, including `host`, `x-amz-date` and
    /// `x-amz-content-sha256`.
    pub headers: &'a [(&'a str, String)],
    /// Hex SHA-256 of the body.
    pub payload_hash: &'a str,
    /// `20261031T030000Z`.
    pub amz_date: &'a str,
}

/// The `Authorization` header for `req`.
pub fn authorization(creds: &Credentials, req: &Request) -> String {
    let query: Vec<String> = req.query.iter().map(|(k, v)| format!("{k}={v}")).collect();
    let headers: String = req
        .headers
        .iter()
        .map(|(k, v)| format!("{k}:{}\n", v.trim()))
        .collect();
    let signed: Vec<&str> = req.headers.iter().map(|(k, _)| *k).collect();
    let signed = signed.join(";");
    let canonical = format!(
        "{}\n{}\n{}\n{headers}\n{signed}\n{}",
        req.method,
        req.path,
        query.join("&"),
        req.payload_hash
    );
    let date = &req.amz_date[..8];
    let scope = format!("{date}/{}/s3/aws4_request", creds.region);
    let to_sign = format!(
        "AWS4-HMAC-SHA256\n{}\n{scope}\n{}",
        req.amz_date,
        sha256_hex(canonical.as_bytes())
    );
    let mut key = hmac(
        format!("AWS4{}", creds.secret_key).as_bytes(),
        date.as_bytes(),
    );
    for part in [creds.region, "s3", "aws4_request"] {
        key = hmac(&key, part.as_bytes());
    }
    format!(
        "AWS4-HMAC-SHA256 Credential={}/{scope}, SignedHeaders={signed}, Signature={}",
        creds.access_key,
        hex::encode(hmac(&key, to_sign.as_bytes()))
    )
}

pub fn sha256_hex(data: &[u8]) -> String {
    hex::encode(Sha256::digest(data))
}

/// URI-encodes `s` as SigV4 wants: everything but `A-Za-z0-9-_.~`, and `/`
/// too unless `keep_slash`.
pub fn encode(s: &str, keep_slash: bool) -> String {
    let mut out = String::with_capacity(s.len());
    for b in s.bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(b as char)
            }
            b'/' if keep_slash => out.push('/'),
            _ => out.push_str(&format!("%{b:02X}")),
        }
    }
    out
}

fn hmac(key: &[u8], data: &[u8]) -> Vec<u8> {
    let mut mac = Hmac::<Sha256>::new_from_slice(key).expect("HMAC takes any key length");
    mac.update(data);
    mac.finalize().into_bytes().to_vec()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn signs_the_aws_example_request() {
        // "Example: GET Object" in the SigV4 header-based auth docs.
        let empty = sha256_hex(b"");
        let headers = [
            ("host", "examplebucket.s3.amazonaws.com".to_string()),
            ("range", "bytes=0-9".to_string()),
            ("x-amz-content-sha256", empty.clone()),
            ("x-amz-date", "20130524T000000Z".to_string()),
        ];
        let creds = Credentials {
            access_key: "AKIAIOSFODNN7EXAMPLE",
            secret_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
            region: "us-east-1",
        };
        let req = Request {
            method: "GET",
            path: "/test.txt",
            query: &[],
            headers: &headers,
            payload_hash: &empty,
            amz_date: "20130524T000000Z",
        };
        assert_eq!(
            authorization(&creds, &req),
            "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, \
             SignedHeaders=host;range;x-amz-content-sha256;x-amz-date, \
             Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
        );
    }

    #[test]
    fn encodes_like_sigv4() {
        assert_eq!(encode("shop/a b+c~.zip", true), "shop/a%20b%2Bc~.zip");
        assert_eq!(encode("shop/x", false), "shop%2Fx");
    }
}
