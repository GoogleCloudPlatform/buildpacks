import unittest

from gcp_buildpack.builder_layer import BuilderLayer

class TestBuilderLayer(unittest.TestCase):
    def test_cache_layer(self):
        test_cases: List[Tuple[str, str]] = [
            ("", True),
            ("0", True),
            ("1", False),
        ]
        for tc in test_cases:
            with self.subTest(name=tc[0]):
                if tc[0]:
                    BuilderLayer.cache = "1"
                layer = BuilderLayer()
                CacheLayer(layer)
                self.assertEqual(layer.cache, tc[1])
