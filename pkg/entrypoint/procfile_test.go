import pytest
from io import StringIO

from pkg.entrypoint.procfile import parse_procfile
from gcpbuildpack.context import Context

test_cases = [
    {
        "name": "empty_file",
        "content": "",
        "want_error": True,
    },
    {
        "name": "one_process",
        "content": "web: bundle exec rackup",
        "want": {"web": "bundle exec rackup"},
    },
    {
        "name": "multiple_processes",
        "content": "web: bundle exec rackup\nworker: bundle exec rake jobs",
        "want": {"web": "bundle exec rackup", "worker": "bundle exec rake jobs"},
    },
    {
        "name": "extra_whitespace",
        "content": "web:    bundle exec rackup   ",
        "want": {"web": "bundle exec rackup"},
    },
    {
        "name": "duplicate_process_keeps_first",
        "content": "web: command1\nweb: command2",
        "want": {"web": "command1"},
        "want_warning": "Skipping duplicate web process: command2",
    },
    {
        "name": "empty_lines_and_comments",
        "content": "# comment\n\nweb: command1",
        "want": {"web": "command1"},
    },
    {
        "name": "empty_lines_and_comments_only",
        "content": "# comment\n\n",
        "want_error": True,
    },
]

@pytest.mark.parametrize("case", test_cases)
def test_parse_procfile(case):
    ctx = Context()
    content = case["content"]
    
    if case.get("want_error"):
        with pytest.raises(UserError):
            parse_procfile(ctx, content)
        return
    
    processes = parse_procfile(ctx, content)
    
    # Check the result
    assert processes == case["want"], f"Test {case['name']} failed: expected {case['want']}, got {processes}"
    
    # Check warnings if applicable
    want_warnings = case.get("want_warning")
    actual_warnings = [rec.message for rec in ctx.logger.handlers[0].buffer]
    
    if want_warnings is not None:
        assert len(actual_warnings) == 1 and actual_warnings[0] == want_warnings, f"Expected warning '{want_warnings}', got {actual_warnings}"
    else:
        assert not actual_warnings, "Unexpected warnings were logged"
