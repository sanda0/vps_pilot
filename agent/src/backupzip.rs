//! Writes a backup zip in the backup child, as the config file's owner: the
//! declared folders and env file under `files/`, the database dump under
//! `database/` and a `manifest.json`. See "Backing up" in docs/PROTOCOL.md.

/// Unix seconds as `20261031T030000Z`, the time in a zip's name.
pub fn stamp(secs: u64) -> String {
    let (y, mo, d, h, mi, s) = civil_time(secs);
    format!("{y:04}{mo:02}{d:02}T{h:02}{mi:02}{s:02}Z")
}

/// The reverse of `stamp`.
pub fn parse_stamp(s: &str) -> Option<u64> {
    let b = s.as_bytes();
    if b.len() != 16 || b[8] != b'T' || b[15] != b'Z' {
        return None;
    }
    let num = |r: std::ops::Range<usize>| s.get(r)?.parse::<u32>().ok();
    let (y, mo, d) = (num(0..4)?, num(4..6)?, num(6..8)?);
    let (h, mi, sec) = (num(9..11)?, num(11..13)?, num(13..15)?);
    if !(1..=12).contains(&mo) || !(1..=31).contains(&d) || h > 23 || mi > 59 || sec > 59 {
        return None;
    }
    let days = days_from_civil(y as i64, mo as i64, d as i64);
    u64::try_from(days * 86400 + (h * 3600 + mi * 60 + sec) as i64).ok()
}

/// UTC (year, month, day, hour, minute, second) of Unix seconds.
fn civil_time(secs: u64) -> (u32, u8, u8, u8, u8, u8) {
    let (days, rem) = ((secs / 86400) as i64, secs % 86400);
    // Howard Hinnant's civil_from_days.
    let z = days + 719_468;
    let (era, doe) = (z.div_euclid(146_097), z.rem_euclid(146_097));
    let yoe = (doe - doe / 1460 + doe / 36_524 - doe / 146_096) / 365;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = yoe + era * 400 + i64::from(m <= 2);
    let (h, mi, s) = (rem / 3600, rem % 3600 / 60, rem % 60);
    (y as u32, m as u8, d as u8, h as u8, mi as u8, s as u8)
}

fn days_from_civil(y: i64, m: i64, d: i64) -> i64 {
    let y = if m <= 2 { y - 1 } else { y };
    let era = y.div_euclid(400);
    let yoe = y - era * 400;
    let doy = (153 * ((m + 9) % 12) + 2) / 5 + d - 1;
    era * 146_097 + yoe * 365 + yoe / 4 - yoe / 100 + doy - 719_468
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stamps_round_trip() {
        for (secs, want) in [
            (0, "19700101T000000Z"),
            (951_782_400, "20000229T000000Z"),
            (1_793_415_600, "20261031T030000Z"),
        ] {
            assert_eq!(stamp(secs), want);
            assert_eq!(parse_stamp(want), Some(secs));
        }
        for bad in ["20261031T030000", "20261331T030000Z", "2026103xT030000Z"] {
            assert_eq!(parse_stamp(bad), None, "{bad}");
        }
    }
}
