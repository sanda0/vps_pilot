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

/// A minute of local time, as `matches` needs it.
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct LocalMinute {
    pub minute: u32,
    pub hour: u32,
    /// 1 ..= 31
    pub day: u32,
    /// 1 ..= 12
    pub month: u32,
    /// 0 ..= 6, Sunday is 0
    pub weekday: u32,
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

    pub fn matches(&self, t: LocalMinute) -> bool {
        let day = self.days & (1 << t.day) != 0;
        let weekday = self.weekdays & (1 << t.weekday) != 0;
        let day_ok = if self.any_day || self.any_weekday {
            day && weekday
        } else {
            day || weekday
        };
        self.minutes & (1 << t.minute) != 0
            && self.hours & (1 << t.hour) != 0
            && self.months & (1 << t.month) != 0
            && day_ok
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

    fn at(minute: u32, hour: u32, day: u32, month: u32, weekday: u32) -> LocalMinute {
        LocalMinute {
            minute,
            hour,
            day,
            month,
            weekday,
        }
    }

    fn runs(expr: &str, t: LocalMinute) -> bool {
        CronSchedule::parse(expr).unwrap().matches(t)
    }

    #[test]
    fn plain_fields() {
        assert!(runs("0 3 * * *", at(0, 3, 15, 6, 2)));
        assert!(!runs("0 3 * * *", at(1, 3, 15, 6, 2)));
        assert!(!runs("0 3 * * *", at(0, 4, 15, 6, 2)));
        assert!(runs("* * * * *", at(59, 23, 31, 12, 6)));
    }

    #[test]
    fn lists_ranges_and_steps() {
        let e = "*/15 9-17 * * 1-5";
        assert!(runs(e, at(45, 17, 3, 11, 5)));
        assert!(!runs(e, at(50, 9, 3, 11, 5)));
        assert!(!runs(e, at(0, 9, 7, 11, 6)));
        assert!(runs("5,10-20/5 * * * *", at(15, 0, 1, 1, 0)));
        assert!(!runs("5,10-20/5 * * * *", at(16, 0, 1, 1, 0)));
    }

    #[test]
    fn sunday_is_0_or_7() {
        assert!(runs("0 0 * * 7", at(0, 0, 1, 11, 0)));
        assert!(runs("0 0 * * 5-7", at(0, 0, 1, 11, 0)));
        assert!(!runs("0 0 * * 7", at(0, 0, 1, 11, 6)));
    }

    #[test]
    fn day_of_month_or_week_when_both_restricted() {
        // The 1st, or any Monday.
        assert!(runs("0 0 1 * 1", at(0, 0, 1, 11, 3)));
        assert!(runs("0 0 1 * 1", at(0, 0, 9, 11, 1)));
        assert!(!runs("0 0 1 * 1", at(0, 0, 9, 11, 2)));
        // A * day of week means only the day of month counts.
        assert!(!runs("0 0 1 * *", at(0, 0, 9, 11, 1)));
    }

    #[test]
    fn macros() {
        assert!(runs("@hourly", at(0, 13, 4, 2, 3)));
        assert!(runs("@daily", at(0, 0, 4, 2, 3)));
        assert!(runs("@weekly", at(0, 0, 8, 11, 0)));
        assert!(!runs("@weekly", at(0, 0, 9, 11, 1)));
        assert!(runs("@monthly", at(0, 0, 1, 7, 4)));
    }

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
