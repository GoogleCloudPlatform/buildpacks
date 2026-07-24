"""
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

import os
import sys
import pathlib

class Testdata:
    @staticmethod
    def must_get_path(relative_path: str) -> str:
        """
        Returns the absolute path of a testdata file.

        This function is designed to work around Blaze's behavior of running tests from the runfiles directory,
        which breaks the standard Go convention of running tests from the test source directory.
        
        Since this function may be called when initializing a package-level variable, it panics on error
        instead of returning an error or requiring the test to pass a *testing.T object.

        :param relative_path: The relative path to the testdata file.
        :return: The absolute path to the testdata file.
        """
        current_dir = pathlib.Path(os.getcwd())
        caller_frame = sys._getframe(1)
        source_file = caller_frame.f_code.co_filename

        if not source_file.endswith("_test.py"):
            raise RuntimeError(f"Invalid caller source file name '{pathlib.Path(source_file).name}': must be invoked from a test")

        return current_dir.joinpath(pathlib.Path(source_file).parent, relative_path)

# No need to explicitly instantiate the Testdata class for this function
must_get_path = Testdata.must_get_path

print(must_get_path("relative/path"))
