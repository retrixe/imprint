package app

import (
	"strconv"
	"strings"
)

type ConfigurationFlags struct {
	UseSystemDd       bool
	DisableValidation bool
}

func ParseCLIFlags(osArgs []string) ([]string, ConfigurationFlags) {
	args := []string{}
	config := ConfigurationFlags{}
	if len(osArgs) == 0 {
		return args, config
	}
	for _, arg := range osArgs[1:] {
		switch arg {
		case "--use-system-dd":
			config.UseSystemDd = true
		case "--disable-validation":
			config.DisableValidation = true
		default:
			args = append(args, arg)
		}
	}
	return args, config
}

func BytesToString(bytes int, binaryPowers bool) string {
	i := ""
	var divisor float64 = 1000
	if binaryPowers {
		i = "i"
		divisor = 1024
	}
	kb := float64(bytes) / divisor
	mb := kb / divisor
	gb := mb / divisor
	tb := gb / divisor
	if tb >= 1 {
		return strconv.FormatFloat(tb, 'f', 1, 64) + " T" + i + "B"
	} else if gb >= 1 {
		return strconv.FormatFloat(gb, 'f', 1, 64) + " G" + i + "B"
	} else if mb >= 1 {
		return strconv.FormatFloat(mb, 'f', 1, 64) + " M" + i + "B"
	} else if kb >= 1 {
		return strconv.FormatFloat(kb, 'f', 1, 64) + " K" + i + "B"
	} else {
		return strconv.Itoa(bytes) + " B"
	}
}

func CapitalizeString(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[0:1]) + str[1:]
}
