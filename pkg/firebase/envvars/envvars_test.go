package envvars

import (
	"fmt"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
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
				"VAR_JSON":           `{"apiKey":"myApiKey","appId":"myAppId"}`,
				"MULTILINE_VAR":      "211 Broadway\nApt. 17\nNew York, NY 10019\n",
			},
			wantEnvMap: map[string]string{
				"API_URL":            "api.service.com",
				"VAR_QUOTED_SPECIAL": "api2.service.com::",
				"VAR_SPACED":         "api3 - service -  com",
				"VAR_SINGLE_QUOTES":  "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":  "\"api4.service.com\"",
				"VAR_NUMBER":         "12345",
				"VAR_JSON":           `{"apiKey":"myApiKey","appId":"myAppId"}`,
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
				"VAR_JSON":           `{"apiKey":"myApiKey","appId":"myAppId"}`,
			},
			wantRawString: `API_URL=api.service.com
MULTILINE_VAR=211 Broadway\nApt. 17\nNew York, NY 10019\n
VAR_DOUBLE_QUOTES="api4.service.com"
VAR_JSON={"apiKey":"myApiKey","appId":"myAppId"}
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

func TestParseEnvVarsFromString(t *testing.T) {
	testCases := []struct {
		desc              string
		serverSideEnvVars string
		wantEnvVars       []apphostingschema.EnvironmentVariable
		wantErr           bool
	}{
		{
			desc: "Parse server side env vars correctly",
			serverSideEnvVars: `
			[
				{
					"Variable": "SERVER_SIDE_ENV_VAR_NUMBER",
					"Value": "3457934845",
					"Availability": ["BUILD", "RUNTIME"]
				},
				{
					"Variable": "SERVER_SIDE_ENV_VAR_MULTILINE_FROM_SERVER_SIDE",
					"Value": "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n",
					"Availability": ["BUILD"]
				},
				{
					"Variable": "SERVER_SIDE_ENV_VAR_QUOTED_SPECIAL",
					"Value": "api_from_server_side.service.com::",
					"Availability": ["RUNTIME"]
				},
				{
					"Variable": "SERVER_SIDE_ENV_VAR_SPACED",
					"Value": "api979 - service -  com",
					"Availability": ["BUILD"]
				},
				{
					"Variable": "SERVER_SIDE_ENV_VAR_SINGLE_QUOTES",
					"Value": "I said, 'I'm learning GOLANG!'",
					"Availability": ["BUILD"]
				},
				{
					"Variable": "SERVER_SIDE_ENV_VAR_DOUBLE_QUOTES",
					"Value": "\"api41.service.com\"",
					"Availability": ["BUILD", "RUNTIME"]
				}
			]
		`,
			wantEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "SERVER_SIDE_ENV_VAR_NUMBER", Value: "3457934845", Availability: []string{"BUILD", "RUNTIME"}},
				{Variable: "SERVER_SIDE_ENV_VAR_MULTILINE_FROM_SERVER_SIDE", Value: "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n", Availability: []string{"BUILD"}},
				{Variable: "SERVER_SIDE_ENV_VAR_QUOTED_SPECIAL", Value: "api_from_server_side.service.com::", Availability: []string{"RUNTIME"}},
				{Variable: "SERVER_SIDE_ENV_VAR_SPACED", Value: "api979 - service -  com", Availability: []string{"BUILD"}},
				{Variable: "SERVER_SIDE_ENV_VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning GOLANG!'", Availability: []string{"BUILD"}},
				{Variable: "SERVER_SIDE_ENV_VAR_DOUBLE_QUOTES", Value: "\"api41.service.com\"", Availability: []string{"BUILD", "RUNTIME"}},
			},
		},
		{
			desc:              "Empty list of server side env vars",
			serverSideEnvVars: "[]",
			wantEnvVars:       []apphostingschema.EnvironmentVariable{},
		},
		{
			desc:              "Malformed server side env vars string",
			serverSideEnvVars: "a malformed string",
			wantEnvVars:       nil,
			wantErr:           true,
		},
	}

	for _, test := range testCases {
		parsedServerSideEnvVars, err := ParseEnvVarsFromString(test.serverSideEnvVars)
		gotErr := err != nil
		if gotErr != test.wantErr {
			t.Errorf("ParseEnvVarsFromString(%q) = %v, want error presence = %v", test.desc, err, test.wantErr)
		}
		if diff := cmp.Diff(test.wantEnvVars, parsedServerSideEnvVars); diff != "" {
			t.Errorf("unexpected env vars for test %q, (+got, -want):\n%v", test.desc, diff)
		}
	}
}
