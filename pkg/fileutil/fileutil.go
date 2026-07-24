import os
import shutil
import tempfile
import unittest
from pathlib import Path

from pkg.fileutil import fileutil

class TestFileUtil(unittest.TestCase):
    def test_maybe_copy_path_contents(self):
        test_cases = [
            {
                "name": "copyAll",
                "app": "testdata/path_with_subdir",
                "condition": lambda path, d: (True, None),
                "want_excluded": []
            },
            {
                "name": "skipFile",
                "app": "testdata/path_with_subdir",
                "condition": lambda path, d: (
                    (False, None) if os.path.basename(path) == "go.mod" else (True, None)
                ),
                "want_excluded": ["subdir/example.com/htmlreturn/go.mod"]
            },
            {
                "name": "skipDir",
                "app": "testdata/path_with_subdir",
                "condition": lambda path, d: (
                    (False, None) if d.is_dir() and os.path.basename(path) == "subdir" else (True, None)
                ),
                "want_excluded": ["subdir"]
            }
        ]

        for case in test_cases:
            with self.subTest(case["name"]):
                tmp_dir = tempfile.mkdtemp()
                src_path = os.path.join(os.path.dirname(__file__), case["app"])
                
                fileutil.MaybeCopyPathContents(tmp_dir, src_path, case["condition"])

                exclude = {".": None}
                for path in case["want_excluded"]:
                    exclude[path] = None

                for root, dirs, files in os.walk(src_path):
                    rel_root = os.path.relpath(root, src_path)
                    
                    if rel_root in exclude:
                        continue
                    
                    for entry in dirs + files:
                        src_entry = os.path.join(root, entry)
                        dest_entry = os.path.join(tmp_dir, os.path.relpath(src_entry, src_path))
                        
                        self.assertTrue(os.path.exists(dest_entry), f"Expected {dest_entry} to exist")

    def test_ensure_unix_line_endings(self):
        test_cases = [
            {
                "name": "no_new_lines",
                "content": "no new lines",
                "want": "no new lines"
            },
            {
                "name": "windows_style_replaced",
                "content": "#!/bin/sh\r\n\r\necho Windows\r\n",
                "want": "#!/bin/sh\n\necho Windows\n"
            },
            {
                "name": "unix_style_unmodified",
                "content": "#!/bin/sh\n\necho Unix\n",
                "want": "#!/bin/sh\n\necho Unix\n"
            }
        ]

        for case in test_cases:
            with self.subTest(case["name"]):
                temp_dir = tempfile.mkdtemp()
                file_path = os.path.join(temp_dir, "file.txt")
                
                Path(file_path).write_bytes(case["content"].encode())
                
                fileutil.EnsureUnixLineEndings(file_path)
                
                content = Path(file_path).read_text()
                self.assertEqual(content, case["want"])

    def test_copy_file(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            src = os.path.join(temp_dir, "src.txt")
            dest = os.path.join(temp_dir, "dest.txt")
            
            content = "hello world"
            Path(src).write_text(content)
            
            fileutil.CopyFile(dest, src)
            
            self.assertEqual(Path(dest).read_text(), content)

if __name__ == "__main__":
    unittest.main()
