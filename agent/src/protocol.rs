//! Wire protocol shared with the hub. See docs/PROTOCOL.md.
//!
//! Each frame is a 4-byte big-endian length followed by a JSON envelope.

use anyhow::{Context, Result, bail};
use serde::{Deserialize, Serialize};
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};

/// Frames larger than this are rejected (matches the hub).
pub const MAX_FRAME_SIZE: u32 = 1 << 20;

pub const MSG_ENROLL: &str = "enroll";
pub const MSG_ENROLLED: &str = "enrolled";
pub const MSG_CONNECTED: &str = "connected";
pub const MSG_AUTH_FAILED: &str = "auth_failed";
pub const MSG_SYS_STAT: &str = "sys_stat";

/// Frame envelope.
#[derive(Debug, Default, Clone, PartialEq, Serialize, Deserialize)]
pub struct Msg {
    #[serde(rename = "type")]
    pub kind: String,
    #[serde(default, skip_serializing_if = "is_zero")]
    pub node_id: i64,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub token: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,
}

fn is_zero(n: &i64) -> bool {
    *n == 0
}

impl Msg {
    pub fn new(kind: &str) -> Self {
        Msg {
            kind: kind.to_string(),
            ..Default::default()
        }
    }

    pub fn with_token(mut self, token: impl Into<String>) -> Self {
        self.token = token.into();
        self
    }

    pub fn with_data<T: Serialize>(mut self, data: &T) -> Result<Self> {
        self.data = Some(serde_json::to_value(data)?);
        Ok(self)
    }

    /// Decodes the data payload into T.
    pub fn payload<T: for<'de> Deserialize<'de>>(&self) -> Result<T> {
        let data = self.data.clone().context("message has no data")?;
        Ok(serde_json::from_value(data)?)
    }
}

/// Sent by the hub in `enrolled`.
#[derive(Debug, Clone, Deserialize)]
pub struct EnrolledPayload {
    pub agent_id: String,
    pub node_key: String,
    pub node_id: i64,
}

/// Sent by the hub in `auth_failed`.
#[derive(Debug, Clone, Default, Deserialize)]
pub struct AuthFailedPayload {
    #[serde(default)]
    pub reason: String,
}

pub async fn read_frame<R: AsyncRead + Unpin>(r: &mut R) -> Result<Msg> {
    let size = r.read_u32().await?;
    if size == 0 || size > MAX_FRAME_SIZE {
        bail!("invalid frame size {size}");
    }
    let mut buf = vec![0u8; size as usize];
    r.read_exact(&mut buf).await?;
    serde_json::from_slice(&buf).context("decode frame")
}

pub async fn write_frame<W: AsyncWrite + Unpin>(w: &mut W, msg: &Msg) -> Result<()> {
    let body = serde_json::to_vec(msg)?;
    if body.len() > MAX_FRAME_SIZE as usize {
        bail!("frame too large: {} bytes", body.len());
    }
    let mut buf = Vec::with_capacity(4 + body.len());
    buf.extend_from_slice(&(body.len() as u32).to_be_bytes());
    buf.extend_from_slice(&body);
    w.write_all(&buf).await?;
    w.flush().await?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn frame_round_trip() {
        let msg = Msg::new(MSG_CONNECTED)
            .with_token("id:key")
            .with_data(&serde_json::json!({"os": "linux"}))
            .unwrap();
        let mut buf = Vec::new();
        write_frame(&mut buf, &msg).await.unwrap();
        assert_eq!(&buf[..4], &((buf.len() - 4) as u32).to_be_bytes());

        let got = read_frame(&mut buf.as_slice()).await.unwrap();
        assert_eq!(got, msg);
    }

    #[tokio::test]
    async fn envelope_matches_hub_json() {
        let msg = Msg::new(MSG_SYS_STAT);
        let json = serde_json::to_string(&msg).unwrap();
        assert_eq!(json, r#"{"type":"sys_stat"}"#);

        let hub: Msg = serde_json::from_str(r#"{"type":"sys_stat","node_id":3}"#).unwrap();
        assert_eq!(hub.node_id, 3);
    }

    #[tokio::test]
    async fn rejects_bad_frame_sizes() {
        for size in [0u32, MAX_FRAME_SIZE + 1] {
            let buf = size.to_be_bytes();
            assert!(read_frame(&mut buf.as_slice()).await.is_err());
        }
    }
}
