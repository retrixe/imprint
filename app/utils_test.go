package app_test

import (
	"testing"

	"github.com/retrixe/imprint/app"
)

func TestParseCLIFlags(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedArgs  []string
		expectedFlags app.ConfigurationFlags
	}{
		{"nil arguments (somehow)", []string{}, []string{}, app.ConfigurationFlags{}},
		{"zero arguments", []string{"imprint"}, []string{}, app.ConfigurationFlags{}},
		{"one argument", []string{"imprint", "arg1"}, []string{"arg1"}, app.ConfigurationFlags{}},
		{"two arguments", []string{"imprint", "arg1", "arg2"}, []string{"arg1", "arg2"}, app.ConfigurationFlags{}},
		{"use-system-dd flag", []string{"imprint", "--use-system-dd"}, []string{}, app.ConfigurationFlags{UseSystemDd: true}},
		{"disable-validation flag", []string{"imprint", "--disable-validation"}, []string{}, app.ConfigurationFlags{DisableValidation: true}},
		{"both flags", []string{"imprint", "--use-system-dd", "--disable-validation"}, []string{}, app.ConfigurationFlags{UseSystemDd: true, DisableValidation: true}},
		{"both flags and arguments", []string{"imprint", "--use-system-dd", "--disable-validation", "arg1", "arg2"}, []string{"arg1", "arg2"}, app.ConfigurationFlags{UseSystemDd: true, DisableValidation: true}},
		{"mixed order of both flags and arguments", []string{"imprint", "arg1", "--use-system-dd", "arg2", "--disable-validation"}, []string{"arg1", "arg2"}, app.ConfigurationFlags{UseSystemDd: true, DisableValidation: true}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			args, config := app.ParseCLIFlags(testCase.args)
			if len(args) != len(testCase.expectedArgs) {
				t.Errorf("expected %d arguments, got %d", len(testCase.expectedArgs), len(args))
			} else if config != testCase.expectedFlags {
				t.Errorf("expected flags %v, got %v", testCase.expectedFlags, config)
			}
		})
	}
}

func TestBytesToString(t *testing.T) {
	testCases := []struct {
		name         string
		bytes        int
		binaryPowers bool
		expected     string
	}{
		{"zero bytes", 0, false, "0 B"},
		{"one byte", 1, false, "1 B"},
		{"one kilobyte", 1000, false, "1.0 KB"},
		{"one megabyte", 1000000, false, "1.0 MB"},
		{"one gigabyte", 1000000000, false, "1.0 GB"},
		{"one terabyte", 1000000000000, false, "1.0 TB"},
		{"one kibibyte", 1024, true, "1.0 KiB"},
		{"one mebibyte", 1048576, true, "1.0 MiB"},
		{"one gibibyte", 1073741824, true, "1.0 GiB"},
		{"one tebibyte", 1099511627776, true, "1.0 TiB"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := app.BytesToString(testCase.bytes, testCase.binaryPowers)
			if result != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, result)
			}
		})
	}
}

func TestCapitalizeString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"capitalized string of length 1", "A", "A"},
		{"capitalized string of length 2", "Aa", "Aa"},
		{"capitalized string of length 3", "AaA", "AaA"},
		{"lowercase string of length 1", "a", "A"},
		{"lowercase string of length 2", "aa", "Aa"},
		{"lowercase string of length 3", "aaa", "Aaa"},
		{"string prefixed with number of length 1", "1", "1"},
		{"string prefixed with number of length 2", "1a", "1a"},
		{"string prefixed with number of length 3", "1ab", "1ab"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := app.CapitalizeString(testCase.input)
			if result != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, result)
			}
		})
	}
}
