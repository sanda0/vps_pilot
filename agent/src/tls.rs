//! TLS to the hub with certificate pinning.
//!
//! The hub uses a self-signed certificate. Instead of a CA chain, the agent
//! pins the SHA-256 fingerprint of the hub's certificate, which it learns from
//! the join token. The handshake signature is still verified, so only the
//! holder of the pinned certificate's private key can complete it.

use std::sync::Arc;

use anyhow::{Context, Result, bail};
use rustls::client::danger::{HandshakeSignatureValid, ServerCertVerified, ServerCertVerifier};
use rustls::crypto::{CryptoProvider, verify_tls12_signature, verify_tls13_signature};
use rustls::pki_types::{CertificateDer, ServerName, UnixTime};
use rustls::{ClientConfig, DigitallySignedStruct, SignatureScheme};
use sha2::{Digest, Sha256};
use tokio::net::TcpStream;
use tokio_rustls::TlsConnector;
use tokio_rustls::client::TlsStream;

/// Name presented in SNI. The hub certificate is not validated by name.
const HUB_SERVER_NAME: &str = "vps-pilot-hub";

/// Lowercase hex SHA-256 of a DER certificate (same as the hub computes).
pub fn fingerprint(der: &[u8]) -> String {
    hex::encode(Sha256::digest(der))
}

/// Parses and normalises a hex fingerprint.
pub fn parse_fingerprint(s: &str) -> Result<String> {
    let s = s.trim().to_ascii_lowercase();
    if s.len() != 64 || !s.bytes().all(|b| b.is_ascii_hexdigit()) {
        bail!("invalid hub fingerprint");
    }
    Ok(s)
}

#[derive(Debug)]
struct PinnedCertVerifier {
    fingerprint: String,
    provider: Arc<CryptoProvider>,
}

impl ServerCertVerifier for PinnedCertVerifier {
    fn verify_server_cert(
        &self,
        end_entity: &CertificateDer<'_>,
        _intermediates: &[CertificateDer<'_>],
        _server_name: &ServerName<'_>,
        _ocsp_response: &[u8],
        _now: UnixTime,
    ) -> Result<ServerCertVerified, rustls::Error> {
        let got = fingerprint(end_entity.as_ref());
        // Fingerprints are public, so a plain comparison is fine here.
        if got == self.fingerprint {
            Ok(ServerCertVerified::assertion())
        } else {
            Err(rustls::Error::General(format!(
                "hub certificate fingerprint mismatch (got {got}, pinned {})",
                self.fingerprint
            )))
        }
    }

    fn verify_tls12_signature(
        &self,
        message: &[u8],
        cert: &CertificateDer<'_>,
        dss: &DigitallySignedStruct,
    ) -> Result<HandshakeSignatureValid, rustls::Error> {
        verify_tls12_signature(
            message,
            cert,
            dss,
            &self.provider.signature_verification_algorithms,
        )
    }

    fn verify_tls13_signature(
        &self,
        message: &[u8],
        cert: &CertificateDer<'_>,
        dss: &DigitallySignedStruct,
    ) -> Result<HandshakeSignatureValid, rustls::Error> {
        verify_tls13_signature(
            message,
            cert,
            dss,
            &self.provider.signature_verification_algorithms,
        )
    }

    fn supported_verify_schemes(&self) -> Vec<SignatureScheme> {
        self.provider
            .signature_verification_algorithms
            .supported_schemes()
    }
}

/// Builds a TLS 1.3 client config that only trusts the pinned certificate.
pub fn client_config(pinned_fingerprint: &str) -> Result<Arc<ClientConfig>> {
    let provider = Arc::new(rustls::crypto::ring::default_provider());
    let verifier = PinnedCertVerifier {
        fingerprint: parse_fingerprint(pinned_fingerprint)?,
        provider: provider.clone(),
    };
    let config = ClientConfig::builder_with_provider(provider)
        .with_protocol_versions(&[&rustls::version::TLS13])?
        .dangerous()
        .with_custom_certificate_verifier(Arc::new(verifier))
        .with_no_client_auth();
    Ok(Arc::new(config))
}

/// Upgrades a TCP connection to TLS against the pinned hub certificate.
pub async fn connect(tcp: TcpStream, config: Arc<ClientConfig>) -> Result<TlsStream<TcpStream>> {
    let name = ServerName::try_from(HUB_SERVER_NAME)?;
    TlsConnector::from(config)
        .connect(name, tcp)
        .await
        .context("TLS handshake with hub")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fingerprint_is_hex_sha256() {
        assert_eq!(
            fingerprint(b"abc"),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
    }

    #[test]
    fn parse_fingerprint_validates() {
        let fp = "AB".repeat(32);
        assert_eq!(parse_fingerprint(&fp).unwrap(), "ab".repeat(32));
        assert!(parse_fingerprint("abc").is_err());
        assert!(parse_fingerprint(&"zz".repeat(32)).is_err());
    }

    #[test]
    fn client_config_rejects_bad_pin() {
        assert!(client_config("nope").is_err());
        assert!(client_config(&"a".repeat(64)).is_ok());
    }
}
