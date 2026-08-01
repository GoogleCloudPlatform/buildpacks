import unittest

from gcp_buildpack.builder_span import BuilderSpan
from gcp_buildpack.fake_exit_handler import FakeExitHandler

class TestNewSpanValidation(unittest.TestCase):
    def test_new_span_name(self):
        bad_name = ""
        with self.assertRaises(ValueError):
            BuilderSpan(name=bad_name, start=time.time(), end=time.time())

    def test_new_span_start_end(self):
        name = "End before start"
        start = time.time()
        end = start - 1
        with self.assertRaises(ValueError):
            BuilderSpan(name=name, start=start, end=end)

class TestCreateSpanName(unittest.TestCase):
    def test_create_span_name(self):
        ctx = BuilderContext()
        test_cases: List[Tuple[str, str]] = [
            ("single", 'Exec "single"'),
            ("one two", 'Exec "one two"'),
            ("invoke $hello", 'Exec "invoke $hello"'),
            ("invoke >pipe", 'Exec "invoke >pipe"'),
            ("invoke --flag && another", 'Exec "invoke --flag && another"'),
            ('echo "DOUBLE QUOTES"', 'Exec "echo \"DOUBLE QUOTES\""'),
            ('echo \'SINGLE QUOTES\'', 'Exec "echo \'SINGLE QUOTES\'"'),
            ("test \r\n\t   characters", 'Exec "test characters"'),
        ]
        for tc in test_cases:
            with self.subTest(name=tc[0]):
                ctx.create_span_name(tc[0])
                expected = tc[1]
                self.assertEqual(expected, BuilderSpan.name)
