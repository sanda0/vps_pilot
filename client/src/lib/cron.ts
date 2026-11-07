// Plain words for common cron schedules (see "Cron jobs" in
// docs/PROTOCOL.md), or null when the expression says it better.

const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
const MACROS: Record<string, string> = {
  "@hourly": "Every hour",
  "@daily": "Every day at 00:00",
  "@weekly": "Sundays at 00:00",
  "@monthly": "On the 1st at 00:00",
};

const isNumber = (s: string) => /^\d+$/.test(s);
const pad = (s: string) => s.padStart(2, "0");

export function describeSchedule(expr: string): string | null {
  const s = expr.trim();
  if (MACROS[s]) return MACROS[s];
  const f = s.split(/\s+/);
  if (f.length !== 5) return null;
  const [minute, hour, day, month, weekday] = f;
  if (day !== "*" || month !== "*") return null;
  if (hour === "*" && weekday === "*") {
    if (minute === "*") return "Every minute";
    const every = minute.match(/^\*\/(\d+)$/);
    if (every) return every[1] === "1" ? "Every minute" : `Every ${every[1]} minutes`;
    if (isNumber(minute)) return `Every hour at :${pad(minute)}`;
    return null;
  }
  if (!isNumber(minute) || !isNumber(hour)) return null;
  const at = `${pad(hour)}:${pad(minute)}`;
  if (weekday === "*") return `Every day at ${at}`;
  if (weekday === "1-5") return `Weekdays at ${at}`;
  if (isNumber(weekday) && Number(weekday) <= 7) return `${DAYS[Number(weekday)]}s at ${at}`;
  return null;
}
