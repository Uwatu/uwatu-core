package config

import (
	"fmt"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

func getTS() string {
	return time.Now().Format("15:04:05")
}

func LogInfo(prefix, message string) {
	fmt.Printf("%s%s %s[INFO] [%-7s]%s %s\n", ColorGray, getTS(), ColorCyan, prefix, ColorReset, message)
}

func LogSuccess(prefix, message string) {
	fmt.Printf("%s%s %s[OK]   [%-7s]%s %s\n", ColorGray, getTS(), ColorGreen, prefix, ColorReset, message)
}

func LogError(prefix, message string) {
	fmt.Printf("%s%s %s[ERR]  [%-7s]%s %s\n", ColorGray, getTS(), ColorRed, prefix, ColorReset, message)
}
