package models

// TagTelemetry maps to the "firmware_payload" object sent by the simulator.
type TagTelemetry struct {
	Seq            int     `json:"seq"`
	AccelMagnitude int     `json:"accel_magnitude"`
	BodyTempC      float64 `json:"body_temp_c"`
	BatteryMv      int     `json:"battery_mv"`
	BatteryPct     int     `json:"battery_pct"`
	UptimeS        int     `json:"uptime_s"`
	SimTrayEvent   bool    `json:"sim_tray_event"`
	RssiDbm        int     `json:"rssi_dbm"`
	CellID         string  `json:"cell_id"`
	Lac            string  `json:"lac"`
	Rat            string  `json:"rat"`
}
