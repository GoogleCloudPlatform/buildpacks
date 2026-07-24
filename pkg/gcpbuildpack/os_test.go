import unittest

from gcp_buildpack.builder_context import BuilderContext

class TestIsWritable(unittest.TestCase):
    def test_is_writable(self):
        test_cases: List[Tuple[int, bool]] = [
            (0o777, True),
            (0o700, True),
            (0o600, False),
            (0o400, False),
            (0o300, True),
            (0o200, False),
            (0o100, False),
        ]
        for tc in test_cases:
            with self.subTest(name=tc[0]):
                f, err := os.CreateTemp(None, "file_")
                if err != nil:
                    raise
                try:
                    f.chmod(tc[0])
                    context = BuilderContext()
                    found, err = context.is_writable(f.name)
                    self.assertTrue(found)
                    self.assertIsNone(err)
                finally:
                    f.close()

    def test_is_not_writable(self):
        test_cases: List[Tuple[int, bool]] = [
            (0o400, False),
            (0o600, True),
            (0o700, True),
            (0o777, True),
        ]
        for tc in test_cases:
            with self.subTest(name=tc[0]):
                f, err := os.CreateTemp(None, "file_")
                if err != nil:
                    raise
                try:
                    f.chmod(tc[0])
                    context = BuilderContext()
                    found, err = context.is_writable(f.name)
                    self.assertFalse(found)
                    self.assertIsNone(err)
                finally:
                    f.close()
