//! A small S3 client for the backup child: upload a zip, list and delete
//! old ones. Works with any S3-compatible storage (Linode Object Storage,
//! AWS, MinIO). See "Backing up" in docs/PROTOCOL.md.

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
    fn reads_s3_xml() {
        let err = "<?xml version=\"1.0\"?><Error><Code>AccessDenied</Code><Message>Access &amp; denied</Message></Error>";
        assert_eq!(error_message(err).unwrap(), "AccessDenied: Access & denied");
        assert_eq!(error_message("<InitiateMultipartUploadResult/>"), None);
        let list = "<ListBucketResult><Contents><Key>a.zip</Key></Contents><Contents><Key>b&amp;c.zip</Key></Contents></ListBucketResult>";
        assert_eq!(tags(list, "Key"), ["a.zip", "b&c.zip"]);
    }
}
