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

// Extended attributes
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

func getTS() string {
	return time.Now().Format("15:04:05")
}

func LogInfo(prefix, message string) {
	fmt.Printf("%s%s %s[INFO] [%-7s]%s %s\n",
		ColorGray, getTS(), ColorCyan, prefix, ColorReset, message)
}

func LogSuccess(prefix, message string) {
	fmt.Printf("%s%s %s[OK]   [%-7s]%s %s\n",
		ColorGray, getTS(), ColorGreen, prefix, ColorReset, message)
}

func LogError(prefix, message string) {
	fmt.Printf("%s%s %s[ERR]  [%-7s]%s %s\n",
		ColorGray, getTS(), ColorRed, prefix, ColorReset, message)
}

// LogEnrich prints a richly formatted telemetry line with conditional colouring.
func LogEnrich(deviceID string, temp float64, accel int, battery int, lat, lon float64,
	simSwapped bool, deviceSwapped bool, roaming bool, reachable string, congestion string) {

	// TAMPER: combines SIM Swap and Device Swap
	tamperStatus := bold + ColorGreen + "OK" + reset
	if simSwapped || deviceSwapped {
		tamperStatus = bold + bgRed + "ALERT" + reset
	}

	// ROAMING
	roamStatus := dim + "---" + reset
	if roaming {
		roamStatus = bold + bgRed + "ROAM" + reset
	}

	// NET: reachability
	reachCode := dim + "---" + reset
	switch reachable {
	case "REACHABLE_DATA":
		reachCode = bold + ColorGreen + "DAT" + reset
	case "REACHABLE_SMS":
		reachCode = bold + ColorYellow + "SMS" + reset
	case "UNREACHABLE":
		reachCode = bold + ColorRed + "OFF" + reset
	}

	// CGEST: congestion
	congestStatus := dim + "---" + reset
	if congestion == "High" {
		congestStatus = bold + ColorRed + "HIGH" + reset
	} else if congestion == "Medium" {
		congestStatus = bold + ColorYellow + "MED" + reset
	} else if congestion == "Low" {
		congestStatus = dim + "LOW" + reset
	}

	// TEMP: fever detection
	tempColor := ColorReset
	if temp > 39.5 {
		tempColor = ColorRed
	} else if temp < 37.0 {
		tempColor = ColorYellow
	}

	// ACCEL: high movement
	accelColor := ColorReset
	if accel > 60 {
		accelColor = ColorRed
	} else if accel > 30 {
		accelColor = ColorYellow
	}

	// BATT: low battery
	battColor := ColorGreen
	if battery < 20 {
		battColor = ColorRed
	} else if battery < 50 {
		battColor = ColorYellow
	}

	// LOC: dim if zero
	locColor := ColorReset
	latStr := fmt.Sprintf("%8.4f", lat)
	lonStr := fmt.Sprintf("%8.4f", lon)
	if lat == 0 && lon == 0 {
		locColor = dim
	}

	// Output
	fmt.Printf(
		"%s%s %s[ENRICH]%s %s%-8s%s %s│%s %sTEMP:%s %s%4.1f°C%s  %sACC:%s %s%3d%s  %sBATT:%s %s%3d%%%s  %sLOC:%s %s%s,%s%s  %sTAMPER:%s %s  %sROAM:%s %s  %sNET:%s %s  %sCGEST:%s %s\n",
		ColorGray, getTS(),
		bold+brightCyan, reset,
		bold, deviceID, reset,
		dim, reset,
		dim, reset, bold+tempColor, temp, reset,
		dim, reset, bold+accelColor, accel, reset,
		dim, reset, bold+battColor, battery, reset,
		dim, reset, bold+locColor, latStr, lonStr, reset,
		dim, reset, tamperStatus,
		dim, reset, roamStatus,
		dim, reset, reachCode,
		dim, reset, congestStatus,
	)
}
