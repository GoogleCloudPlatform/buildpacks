import glob
import os
from pathlib import Path
from typing import Optional, List, Callable, Any, Union

class Context:
    def __init__(self):
        self.application_root = ""

    @property
    def ApplicationRoot(self) -> str:
        return self.application_root

    def Glob(self, pattern: str) -> Union[List[str], Exception]:
        matches = []
        try:
            matches = glob.glob(os.path.join(self.ApplicationRoot, pattern))
        except Exception as e:
            return buildererror.Error(buildererror.Status.Internal, f"globbing {pattern}: {e}")
        return matches

    def HasAtLeastOne(self, pattern: str) -> Union[bool, Exception]:
        return self.HasAtLeastOneFiltered(pattern, None)

    def HasAtLeastOneOutsideDependencyDirectories(self, pattern: str) -> Union[bool, Exception]:
        filter_func = lambda path: not path.endswith("/node_modules")
        return self.HasAtLeastOneFiltered(pattern, filter_func)

    def HasAtLeastOneFiltered(
        self,
        pattern: str,
        filter_func: Optional[Callable[[str], bool]]
    ) -> Union[bool, Exception]:
        dir_path = os.path.join(self.ApplicationRoot)
        matches = glob.glob(os.path.join(dir_path, pattern))
        if len(matches) > 0:
            return True

        try:
            for root, dirs, files in os.walk(dir_path):
                for name in files + dirs:
                    full_path = os.path.join(root, name)
                    if filter_func and not filter_func(full_path):
                        continue
                    match = self._match_pattern(name, pattern)
                    if match:
                        return True
        except Exception as e:
            return buildererror.Error(buildererror.Status.Internal, f"walking through {dir_path}: {e}")

        return False

    def _match_pattern(self, name: str, pattern: str) -> bool:
        try:
            return glob.fnmatch.fnmatch(name, pattern)
        except Exception as e:
            raise buildererror.Error(
                buildererror.Status.Internal,
                f"matching {name} with pattern {pattern}: {e}"
            )
