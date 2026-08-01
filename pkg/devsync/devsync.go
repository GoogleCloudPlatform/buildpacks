import os
import unittest
from pathlib import Path

class TestDevSync(unittest.TestCase):
    def test_update_app_run_script(self):
        with tempfile.TemporaryDirectory() as dir:
            app_dir = os.path.join(dir, "app")
            os.makedirs(app_dir, exist_ok=True)
            
            template_path = os.path.join(app_dir, "run.template")
            with open(template_path, 'w') as f:
                f.write("#!/bin/bash\ncd /workspace || exit 1\nexec chpst -e ./env -P {{ENTRYPOINT}}\n")
            
            env_vars = {"PYTHONPATH": "/custom/path"}
            update_app_run_script(dir, "node --watch server.js", env_vars)
            
            run_path = os.path.join(app_dir, "run")
            with open(run_path, 'r') as f:
                content = f.read()
            
            expected = "#!/bin/bash\ncd /workspace || exit 1\nexec chpst -e ./env -P node --watch server.js\n"
            self.assertEqual(content, expected)
            
            env_val_path = os.path.join(app_dir, "env", "PYTHONPATH")
            with open(env_val_path, 'r') as f:
                val = f.read()
            self.assertEqual(val, "/custom/path")
            
            # Verify template is preserved
            self.assertTrue(os.path.exists(template_path))

if __name__ == '__main__':
    unittest.main()
