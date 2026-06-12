package config

import (
	"os"
	"time"
)

var Timezones = []string{
	"UTC",
	"Europe/London",
	"Europe/Berlin",
	"Europe/Paris",
	"Europe/Warsaw",
	"Europe/Helsinki",
	"Europe/Madrid",
	"Europe/Rome",
	"Europe/Amsterdam",
	"Europe/Stockholm",
	"Europe/Oslo",
	"Europe/Brussels",
	"Europe/Vienna",
	"Europe/Prague",
	"Europe/Budapest",
	"Europe/Bucharest",
	"Europe/Kyiv",
	"Europe/Moscow",
	"Europe/Istanbul",
	"Europe/Lisbon",
	"Europe/Dublin",
	"America/New_York",
	"America/Chicago",
	"America/Denver",
	"America/Los_Angeles",
	"America/Toronto",
	"America/Vancouver",
	"America/Sao_Paulo",
	"America/Buenos_Aires",
	"America/Mexico_City",
	"America/Santiago",
	"America/Lima",
	"America/Bogota",
	"America/Caracas",
	"America/Phoenix",
	"America/Anchorage",
	"America/Halifax",
	"America/Winnipeg",
	"Asia/Tokyo",
	"Asia/Shanghai",
	"Asia/Hong_Kong",
	"Asia/Singapore",
	"Asia/Seoul",
	"Asia/Kolkata",
	"Asia/Dubai",
	"Asia/Bangkok",
	"Asia/Jakarta",
	"Asia/Manila",
	"Asia/Taipei",
	"Asia/Tehran",
	"Asia/Karachi",
	"Asia/Dhaka",
	"Asia/Riyadh",
	"Asia/Jerusalem",
	"Australia/Sydney",
	"Australia/Melbourne",
	"Australia/Brisbane",
	"Australia/Perth",
	"Australia/Adelaide",
	"Pacific/Auckland",
	"Pacific/Fiji",
	"Pacific/Honolulu",
	"Africa/Johannesburg",
	"Africa/Nairobi",
	"Africa/Lagos",
	"Africa/Cairo",
	"Africa/Casablanca",
	"Africa/Addis_Ababa",
	"Atlantic/Azores",
	"Atlantic/Cape_Verde",
}

var normalizeTZ = map[string]string{
	"UTC":   "UTC",
	"GMT":   "UTC",
	"CET":   "Europe/Berlin",
	"CEST":  "Europe/Berlin",
	"EET":   "Europe/Helsinki",
	"EEST":  "Europe/Helsinki",
	"WET":   "Europe/London",
	"WEST":  "Europe/London",
	"EST":   "America/New_York",
	"EDT":   "America/New_York",
	"CST":   "America/Chicago",
	"CDT":   "America/Chicago",
	"MST":   "America/Denver",
	"MDT":   "America/Denver",
	"PST":   "America/Los_Angeles",
	"PDT":   "America/Los_Angeles",
}

func SystemTimezone() string {
	local := time.Local.String()

	if local != "Local" {
		if n, ok := normalizeTZ[local]; ok {
			return n
		}
		return local
	}

	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}

	abbr, _ := time.Now().Zone()
	if n, ok := normalizeTZ[abbr]; ok {
		return n
	}
	return abbr
}

func SystemTimezoneIndex() int {
	tz := SystemTimezone()
	for i, candidate := range Timezones {
		if candidate == tz {
			return i
		}
	}
	return 0
}

