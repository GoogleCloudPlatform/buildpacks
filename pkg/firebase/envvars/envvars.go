import os
import tempfile
from pathlib import Path

import pytest

from pkg.firebase.envvars import (
    write, write_lifecycle, read_lifecycle, read, marshal,
    parse_env_vars_from_string
)
from pkg.firebase.apphostingschema import EnvironmentVariable

def test_write():
    test_dir = tempfile.mkdtemp()
    
    test_cases = [
        {
            'desc': "Write custom env file correctly",
            'input_env_map': {
                "API_URL": "api.service.com",
                "VAR_QUOTED_SPECIAL": "api2.service.com::",
                "VAR_SPACED": "api3 - service -  com",
                "VAR_SINGLE_QUOTES": "I said, 'I'm learning YAML!'",
                "VAR_DOUBLE_QUOTES": "\"api4.service.com\"",
                "VAR_NUMBER": "12345",
                "VAR_JSON": `{"apiKey":"myApiKey","appId":"myAppId"}`,
                "MULTILINE_VAR": "211 Broadway\nApt. 17\nNew York, NY 10019\n"
            },
            'want_env_map': {
                "API_URL": "api.service.com",
                "VAR_QUOTED_SPECIAL": "api2.service.com::",
                "VAR_SPACED": "api3 - service -  com",
                "VAR_SINGLE_QUOTES": "I said, 'I'm learning YAML!'",
                "VAR_DOUBLE_QUOTES": "\"api4.service.com\"",
                "VAR_NUMBER": "12345",
                "VAR_JSON": `{"apiKey":"myApiKey","appId":"myAppId"}`,
                "MULTILINE_VAR": "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n"
            }
        },
        {
            'desc': "Writes file even with an empty map",
            'input_env_map': {},
            'want_env_map': {}
        }
    ]

    for i, test in enumerate(test_cases):
        output_file = os.path.join(test_dir, f'output{i}')
        
        write(test['input_env_map'], output_file)
        
        actual_map = read(output_file)
        assert actual_map == test['want_env_map']

def test_write_lifecycle():
    test_dir = tempfile.mkdtemp()
    
    test_cases = [
        {
            'desc': "Write custom env file correctly",
            'input_env_map': {
                "API_URL": "api.service.com",
                "VAR_QUOTED_SPECIAL": "api2.service.com::",
                "VAR_SPACED": "api3 - service -  com",
                "VAR_SINGLE_QUOTES": "I said, 'I'm learning YAML!'",
                "VAR_DOUBLE_QUOTES": "\"api4.service.com\"",
                "VAR_NUMBER": "12345",
                "VAR_JSON": `{"apiKey":"myApiKey","appId":"myAppId"}`,
                "MULTILINE_VAR": "211 Broadway\nApt. 17\nNew York, NY 10019\n"
            },
            'want_env_map': {
                "API_URL": "api.service.com",
                "VAR_QUOTED_SPECIAL": "api2.service.com::",
                "VAR_SPACED": "api3 - service -  com",
                "VAR_SINGLE_QUOTES": "I said, 'I'm learning YAML!'",
                "VAR_DOUBLE_QUOTES": "\"api4.service.com\"",
                "VAR_NUMBER": "12345",
                "VAR_JSON": `{"apiKey":"myApiKey","appId":"myAppId"}`,
                "MULTILINE_VAR": "211 Broadway\nApt. 17\nNew York, NY 10019\n"
            }
        },
        {
            'desc': "No error when writing empty map",
            'input_env_map': {},
            'want_env_map': {}
        }
    ]

    for i, test in enumerate(test_cases):
        output_dir = os.path.join(test_dir, f'output{i}')
        
        write_lifecycle(test['input_env_map'], output_dir)
        
        actual_map = read_lifecycle(output_dir)
        assert actual_map == test['want_env_map']

def test_write_raw_data():
    test_dir = tempfile.mkdtemp()
    
    test_cases = [
        {
            'desc': "Write custom env file correctly and verify against raw data",
            'input_env_map': {
                "API_URL": "api.service.com",
                "VAR_QUOTED_SPECIAL": "api2.service.com::",
                "VAR_SPACED": "api3 - service -  com",
                "VAR_SINGLE_QUOTES": "I said, 'I'm learning YAML!'",
                "VAR_DOUBLE_QUOTES": "\"api4.service.com\"",
                "VAR_NUMBER": "12345",
                "MULTILINE_VAR": "211 Broadway\nApt. 17\nNew York, NY 10019\n",
                "VAR_JSON": `{"apiKey":"myApiKey","appId":"myAppId"}`
            },
            'want_raw_string': (
                "API_URL=api.service.com\n"
                "MULTILINE_VAR=211 Broadway\\nApt. 17\\nNew York, NY 10019\\n\n"
                "VAR_DOUBLE_QUOTES=\"api4.service.com\"\n"
                "VAR_JSON={\"apiKey\":\"myApiKey\",\"appId\":\"myAppId\"}\n"
                "VAR_NUMBER=12345\n"
                "VAR_QUOTED_SPECIAL=api2.service.com::\n"
                "VAR_SINGLE_QUOTES=I said, 'I'm learning YAML!'\n"
                "VAR_SPACED=api3 - service -  com\n"
            )
        },
        {
            'desc': "Writes raw file properly even with an empty map",
            'input_env_map': {},
            'want_raw_string': "\n"
        }
    ]

    for i, test in enumerate(test_cases):
        output_file = os.path.join(test_dir, f'output{i}')
        
        write(test['input_env_map'], output_file)
        
        with open(output_file, 'r') as f:
            data = f.read()
        assert data == test['want_raw_string']

def test_parse_env_vars_from_string():
    test_cases = [
        {
            'desc': "Parse server side env vars correctly",
            'server_side_env_vars': '''
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
            ''',
            'want_env_vars': [
                EnvironmentVariable(
                    variable="SERVER_SIDE_ENV_VAR_NUMBER",
                    value="3457934845",
                    availability=["BUILD", "RUNTIME"],
                    source='firebase-console'
                ),
                # ... other variables similarly defined
            ]
        },
        {
            'desc': "Empty list of server side env vars",
            'server_side_env_vars': "[]",
            'want_env_vars': []
        },
        {
            'desc': "Malformed server side env vars string",
            'server_side_env_vars': "a malformed string",
            'want_env_vars': None,
            'want_err': True
        }
    ]

    for test in test_cases:
        if test.get('want_err'):
            with pytest.raises(ValueError):
                parse_env_vars_from_string(test['server_side_env_vars'])
        else:
            result = parse_env_vars_from_string(test['server_side_env_vars'])
            assert len(result) == len(test['want_env_vars'])
            for got, want in zip(result, test['want_env_vars']):
                assert got.variable == want.variable
                assert got.value == want.value
                assert got.availability == want.availability
                assert got.source == 'firebase-console'
