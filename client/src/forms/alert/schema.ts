import * as z from "zod"

export interface ActionResponse < T = any > {
  success: boolean
  message: string
  errors ? : {
    [K in keyof T] ? : string[]
  }
  inputs ? : T
}
export const formSchema = z.object({
  "id": z.number().optional(),
  "node_id": z.number().optional(),
  "metric": z.string().min(1),
  "threshold": z.number().optional(),
  "net_rece_threshold": z.number().optional(),
  "net_send_threshold": z.number().optional(),
  "duration": z.number().optional(),
  "email": z.string().optional(),
  "discord": z.string().optional(),
  "slack": z.string().optional(),
  "enabled": z.boolean().optional()
});