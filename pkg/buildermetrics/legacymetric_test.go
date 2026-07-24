import unittest

class TestFieldListsMatch(unittest.TestCase):
    def test_field_lists_match(self):
        l1 = Label("l1", "String")
        l2 = Label("l2", "Bool")
        l3 = Label("l3", "Int")
        f1 = Field(l1, "s1")
        f2 = Field(l2, True)
        f3 = Field(l3, 42)

        test_cases = [
            {"name": "all matching", "fl1": [f1, f2, f3], "fl2": [f1, f2, f3], "want": True},
            {"name": "all matching out of order", "fl1": [f1, f2, f3], "fl2": [f3, f2, f1], "want": True},
            {"name": "missing a field", "fl1": [f1, f2, f3], "fl2": [f3, f2, f1], "want": False},
            {"name": "extra field", "fl1": [f1, f3], "fl2": [f3, f2, f1], "want": False},
            {"name": "left empty", "fl1": [], "fl2": [f1, f2, f3], "want": False},
            {"name": "right empty", "fl1": [f1, f2, f3], "fl2": [], "want": False},
            {"name": "both empty", "fl1": [], "fl2": [], "want": True}
        ]

        for tc in test_cases:
            self.assertTrue(field_lists_match(tc["fl1"], tc["fl2"]) == tc["want"])

if __name__ == "__main__":
    unittest.main()
