//! System info and periodic stats, in the JSON shapes the hub expects.

use std::path::Path;
use std::time::Instant;

use serde::Serialize;
use sysinfo::{Disks, Networks, System};

#[derive(Debug, Clone, Serialize)]
pub struct SystemInfo {
    pub hostname: String,
    pub os: String,
    pub platform: String,
    pub platform_version: String,
    pub kernel_version: String,
    pub cpus: usize,
    /// Bytes.
    pub total_memory: u64,
    pub agent_version: &'static str,
    /// What this agent can do beyond stats and projects (docs/PROTOCOL.md).
    pub features: &'static [&'static str],
}

#[derive(Debug, Clone, Serialize)]
pub struct SystemStat {
    /// Percent per core.
    pub cpu_usage: Vec<f64>,
    /// Percent.
    pub mem_usage: f64,
    /// Percent used of the root filesystem.
    pub disk_usage: f64,
    /// Bytes per second.
    pub net_sent_ps: u64,
    pub net_recv_ps: u64,
}

/// Network byte counters at a point in time.
#[derive(Debug, Clone, Copy)]
struct NetSample {
    at: Instant,
    sent: u64,
    recv: u64,
}

/// Bytes per second between two samples. Counter resets (interface restart)
/// yield 0 rather than a huge bogus value.
fn rate(prev: u64, now: u64, secs: f64) -> u64 {
    if secs <= 0.0 {
        return 0;
    }
    (now.saturating_sub(prev) as f64 / secs) as u64
}

fn percent(part: u64, whole: u64) -> f64 {
    if whole == 0 {
        0.0
    } else {
        part as f64 / whole as f64 * 100.0
    }
}

pub struct Collector {
    sys: System,
    disks: Disks,
    networks: Networks,
    prev_net: Option<NetSample>,
}

impl Collector {
    pub fn new() -> Self {
        let mut sys = System::new();
        // CPU usage is computed between two refreshes; prime the first one.
        sys.refresh_cpu_usage();
        sys.refresh_memory();
        let mut collector = Collector {
            sys,
            disks: Disks::new_with_refreshed_list(),
            networks: Networks::new_with_refreshed_list(),
            prev_net: None,
        };
        collector.prev_net = Some(collector.net_sample());
        collector
    }

    pub fn system_info(&self) -> SystemInfo {
        SystemInfo {
            hostname: System::host_name().unwrap_or_default(),
            os: std::env::consts::OS.to_string(),
            platform: System::distribution_id(),
            platform_version: System::os_version().unwrap_or_default(),
            kernel_version: System::kernel_version().unwrap_or_default(),
            cpus: self.sys.cpus().len(),
            total_memory: self.sys.total_memory(),
            agent_version: crate::VERSION,
            features: &[],
        }
    }

    pub fn collect(&mut self) -> SystemStat {
        self.sys.refresh_cpu_usage();
        self.sys.refresh_memory();
        self.disks.refresh(true);
        self.networks.refresh(true);

        let now = self.net_sample();
        let (net_sent_ps, net_recv_ps) = match self.prev_net {
            Some(prev) => {
                let secs = now.at.duration_since(prev.at).as_secs_f64();
                (
                    rate(prev.sent, now.sent, secs),
                    rate(prev.recv, now.recv, secs),
                )
            }
            None => (0, 0),
        };
        self.prev_net = Some(now);

        SystemStat {
            cpu_usage: self
                .sys
                .cpus()
                .iter()
                .map(|c| c.cpu_usage() as f64)
                .collect(),
            mem_usage: percent(self.sys.used_memory(), self.sys.total_memory()),
            disk_usage: self.root_disk_usage(),
            net_sent_ps,
            net_recv_ps,
        }
    }

    fn root_disk_usage(&self) -> f64 {
        self.disks
            .list()
            .iter()
            .find(|d| d.mount_point() == Path::new("/"))
            .map(|d| {
                percent(
                    d.total_space().saturating_sub(d.available_space()),
                    d.total_space(),
                )
            })
            .unwrap_or(0.0)
    }

    /// Totals across interfaces, excluding loopback.
    fn net_sample(&self) -> NetSample {
        let (sent, recv) = self
            .networks
            .list()
            .iter()
            .filter(|(name, _)| name.as_str() != "lo")
            .fold((0u64, 0u64), |(s, r), (_, data)| {
                (s + data.total_transmitted(), r + data.total_received())
            });
        NetSample {
            at: Instant::now(),
            sent,
            recv,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rate_per_second() {
        assert_eq!(rate(1_000, 11_000, 5.0), 2_000);
        assert_eq!(rate(5_000, 1_000, 5.0), 0, "counter reset");
        assert_eq!(rate(0, 100, 0.0), 0);
    }

    #[test]
    fn percent_handles_zero() {
        assert_eq!(percent(1, 0), 0.0);
        assert_eq!(percent(25, 100), 25.0);
    }

    #[test]
    fn collects_plausible_values() {
        let mut c = Collector::new();
        let info = c.system_info();
        assert!(info.cpus > 0);
        assert!(info.total_memory > 0);

        let stat = c.collect();
        assert_eq!(stat.cpu_usage.len(), info.cpus);
        assert!((0.0..=100.0).contains(&stat.mem_usage));
        assert!((0.0..=100.0).contains(&stat.disk_usage));
    }
}
