package envvars

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWrite(t *testing.T) {
	testDir := t.TempDir()

	testCases := []struct {
		desc        string
		inputEnvMap map[string]string
		wantEnvMap  map[string]string
	}{
		{
			desc: "Write custom env file correctly",
			inputEnvMap: map[string]string{
				"API_URL":            "api.service.com",
				"VAR_QUOTED_SPECIAL": "api2.service.com::",
				"VAR_SPACED":         "api3 - service -  com",
				"VAR_SINGLE_QUOTES":  "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":  "\"api4.service.com\"",
				"VAR_NUMBER":         "12345",
				"MULTILINE_VAR":      "211 Broadway\nApt. 17\nNew York, NY 10019\n",
			},
			wantEnvMap: map[string]string{
				"API_URL":            "api.service.com",
				"VAR_QUOTED_SPECIAL": "api2.service.com::",
				"VAR_SPACED":         "api3 - service -  com",
				"VAR_SINGLE_QUOTES":  "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":  "\"api4.service.com\"",
				"VAR_NUMBER":         "12345",
				// Key difference is that the newline character is now properly escaped
				"MULTILINE_VAR": "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n",
			},
		},
		{
			desc:        "Writes file even with an empty map",
			inputEnvMap: map[string]string{},
			wantEnvMap:  map[string]string{},
		},
	}

	for i, test := range testCases {
		outputFilePath := fmt.Sprintf("%s/output%d", testDir, i)

		err := Write(test.inputEnvMap, outputFilePath)
		if err != nil {
			t.Errorf("error in test '%v'. Error was %v", test.desc, err)
		}

		actualMap, err := Read(outputFilePath)
		if err != nil {
			t.Errorf("error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.wantEnvMap, actualMap); diff != "" {
			t.Errorf("unexpected map for test %q, (+got, -want):\n%v", test.desc, diff)
		}
	}
}

func TestWriteRawData(t *testing.T) {
	testDir := t.TempDir()

	testCases := []struct {
		desc          string
		inputEnvMap   map[string]string
		wantRawString string
	}{
		{
			desc: "Write custom env file correctly and verify against raw data",
			inputEnvMap: map[string]string{
				"API_URL":            "api.service.com",
				"VAR_QUOTED_SPECIAL": "api2.service.com::",
				"VAR_SPACED":         "api3 - service -  com",
				"VAR_SINGLE_QUOTES":  "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":  "\"api4.service.com\"",
				"VAR_NUMBER":         "12345",
				"MULTILINE_VAR":      "211 Broadway\nApt. 17\nNew York, NY 10019\n",
			},
			wantRawString: `API_URL=api.service.com
MULTILINE_VAR=211 Broadway\nApt. 17\nNew York, NY 10019\n
VAR_DOUBLE_QUOTES="api4.service.com"
VAR_NUMBER=12345
VAR_QUOTED_SPECIAL=api2.service.com::
VAR_SINGLE_QUOTES=I said, 'I'm learning YAML!'
VAR_SPACED=api3 - service -  com
`,
		},
		{
			desc:          "Writes raw file properly even with an empty map",
			inputEnvMap:   map[string]string{},
			wantRawString: "\n",
		},
	}

	for i, test := range testCases {
		outputFilePath := fmt.Sprintf("%s/output%d", testDir, i)

		err := Write(test.inputEnvMap, outputFilePath)
		if err != nil {
			t.Errorf("error in test '%v'. Error was %v", test.desc, err)
		}

		data, err := os.ReadFile(outputFilePath)
		if err != nil {
			t.Errorf("error reading file: %v", err)
		}

		if diff := cmp.Diff(test.wantRawString, string(data)); diff != "" {
			t.Errorf("unexpected raw string for test %q, (+got, -want):\n%v", test.desc, diff)
		}
	}
}
