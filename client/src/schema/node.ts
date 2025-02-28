import { z } from "zod";

export const AlertSchema = z.object({
  metric: z.enum(["CPU", "Memory", "Network"], { required_error: "Select a metric" }),
  threshold: z.number().min(0).max(100, "Threshold must be between 0-100"),
  duration: z.number().min(1, "Duration must be at least 1 minute"),
  // email: z.string().email().optional(),
  discord: z.string().url().optional(),
  enabled: z.boolean(),
});