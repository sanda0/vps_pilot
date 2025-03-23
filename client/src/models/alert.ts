export interface Alert {
  id:                 number;
  node_id:            number;
  metric:             string;
  duration:           number;
  threshold:          Threshold;
  net_rece_threshold: Threshold;
  net_send_threshold: Threshold;
  email:              PgString;
  discord_webhook:    PgString;
  slack_webhook:      PgString;
  is_active:          IsActive;
  created_at:         Date;
  updated_at:         Date;
}

export interface PgString {
  String: string;
  Valid:  boolean;
}

export interface IsActive {
  Bool:  boolean;
  Valid: boolean;
}

export interface Threshold {
  Float64: number;
  Valid:   boolean;
}
