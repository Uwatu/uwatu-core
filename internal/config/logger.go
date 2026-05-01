package config

import (
	"fmt"
	"time"
)

// Basic colours
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// Extended text attributes & colours
const (
	reset       = "\033[0m"
	bold        = "\033[1m"
	dim         = "\033[2m"
	underline   = "\033[4m"
	bgRed       = "\033[41m"
	bgGreen     = "\033[42m"
	brightCyan  = "\033[96m"
	brightWhite = "\033[97m"
)

// getTS returns a formatted timestamp (HH:MM:SS)
func getTS() string {
	return time.Now().Format("15:04:05")
}

// LogInfo prints an informational message with a cyan prefix.
func LogInfo(prefix, message string) {
	fmt.Printf("%s%s %s[INFO] [%-7s]%s %s\n",
		ColorGray, getTS(), ColorCyan, prefix, ColorReset, message)
}

// LogSuccess prints a success message with a green OK prefix.
func LogSuccess(prefix, message string) {
	fmt.Printf("%s%s %s[OK]   [%-7s]%s %s\n",
		ColorGray, getTS(), ColorGreen, prefix, ColorReset, message)
}

// LogError prints an error message with a red ERR prefix.
func LogError(prefix, message string) {
	fmt.Printf("%s%s %s[ERR]  [%-7s]%s %s\n",
		ColorGray, getTS(), ColorRed, prefix, ColorReset, message)
}

// LogEnrich prints a clean, bold telemetry line without emoji.
// It expects the device ID, temperature, accelerometer, battery, lat, lon, and sim swapped flag.
func LogEnrich(deviceID string, temp float64, accel int, battery int, lat, lon float64, simSwapped bool) {
	// Format sim status
	simStatus := "OK"
	if simSwapped {
		simStatus = bold + bgRed + "ALERT" + reset
	} else {
		simStatus = bold + "OK" + reset
	}

	fmt.Printf(
		"%s%s %s[ENRICH]%s %s%-8s%s %s│%s %sTEMP:%s %s%4.1f°C%s  %sACC:%s %s%3d%s  %sBATT:%s %s%3d%%%s  %sLOC:%s %s%8.4f, %8.4f%s  %sSIM:%s %s\n",
		ColorGray, getTS(),
		bold+brightCyan, reset,
		bold, deviceID, reset,
		dim, reset,
		dim, reset, bold, temp, reset,
		dim, reset, bold, accel, reset,
		dim, reset, bold, battery, reset,
		dim, reset, bold, lat, lon, reset,
		dim, reset, simStatus,
	)
}
