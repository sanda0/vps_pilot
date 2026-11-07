//! Cron schedules: the five-field `minute hour day-of-month month
//! day-of-week` expressions of `cron` in config.vpspilot.json, plus
//! `@hourly`, `@daily`, `@weekly` and `@monthly`. See "Cron jobs" in
//! docs/PROTOCOL.md.

// Wired into the config check and the scheduler in the next changes.
#![allow(dead_code)]

/// A parsed schedule. Each field is a bit set of the values it matches.
#[derive(Debug, Clone, PartialEq)]
pub struct CronSchedule {
    minutes: u64,
    hours: u32,
    days: u32,
    months: u16,
    weekdays: u8,
    /// The day-of-month or day-of-week field starts with `*`. Only when
    /// both are restricted does a day matching either one count, as in
    /// system cron.
    any_day: bool,
    any_weekday: bool,
}

impl CronSchedule {
    pub fn parse(expr: &str) -> Result<Self, String> {
        let expanded = match expr.trim() {
            "@hourly" => "0 * * * *",
            "@daily" => "0 0 * * *",
            "@weekly" => "0 0 * * 0",
            "@monthly" => "0 0 1 * *",
            s if s.starts_with('@') => return Err(format!("unknown schedule {s:?}")),
            s => s,
        };
        let fields: Vec<&str> = expanded.split_whitespace().collect();
        let [minute, hour, day, month, weekday] = fields[..] else {
            return Err(format!(
                "schedule {expr:?} must have 5 fields: minute hour day-of-month month day-of-week"
            ));
        };
        let weekdays = field(weekday, "day of week", 0, 7)?;
        Ok(CronSchedule {
            minutes: field(minute, "minute", 0, 59)?,
            hours: field(hour, "hour", 0, 23)? as u32,
            days: field(day, "day of month", 1, 31)? as u32,
            months: field(month, "month", 1, 12)? as u16,
            // 7 is Sunday too.
            weekdays: ((weekdays | (weekdays >> 7)) & 0x7f) as u8,
            any_day: day.starts_with('*'),
            any_weekday: weekday.starts_with('*'),
        })
    }
}

/// One field: `*`, `a`, `a-b`, `*/n` or `a-b/n`, or a comma-separated
/// list of those. Returns the matched values as bits.
fn field(s: &str, name: &str, min: u32, max: u32) -> Result<u64, String> {
    let mut bits = 0u64;
    for item in s.split(',') {
        let (range, step) = match item.split_once('/') {
            Some((range, step)) => {
                let step = number(step, name)?;
                if step == 0 {
                    return Err(format!("{name} step must be at least 1"));
                }
                if !range.contains('-') && range != "*" {
                    return Err(format!("{name} {item:?}: a step needs * or a range"));
                }
                (range, step)
            }
            None => (item, 1),
        };
        let (lo, hi) = match range {
            "*" => (min, max),
            _ => match range.split_once('-') {
                Some((lo, hi)) => (number(lo, name)?, number(hi, name)?),
                None => {
                    let v = number(range, name)?;
                    (v, v)
                }
            },
        };
        for v in [lo, hi] {
            if !(min..=max).contains(&v) {
                return Err(format!("{name} {v} is out of range {min}-{max}"));
            }
        }
        if lo > hi {
            return Err(format!("{name} range {range:?} goes backwards"));
        }
        for v in (lo..=hi).step_by(step as usize) {
            bits |= 1 << v;
        }
    }
    Ok(bits)
}

fn number(s: &str, name: &str) -> Result<u32, String> {
    if s.is_empty() || !s.bytes().all(|b| b.is_ascii_digit()) {
        return Err(format!("{name} {s:?} is not a number"));
    }
    s.parse().map_err(|_| format!("{name} {s:?} is too large"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_bad_expressions() {
        for bad in [
            "",
            "* * * *",
            "* * * * * *",
            "60 * * * *",
            "* 24 * * *",
            "* * 0 * *",
            "* * * 13 *",
            "* * * * 8",
            "*/0 * * * *",
            "5/10 * * * *",
            "10-5 * * * *",
            "MON * * * *",
            "* * * JAN *",
            "-1 * * * *",
            "1,,2 * * * *",
            "@yearly",
            "@reboot",
        ] {
            assert!(
                CronSchedule::parse(bad).is_err(),
                "{bad:?} should be rejected"
            );
        }
    }
}
