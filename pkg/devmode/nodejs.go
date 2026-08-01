# Complete refactored code here
"""
Package devmode contains helpers to configure Development Mode.
"""

import os
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional

import shlex
import subprocess

class GCPContext:
    def __init__(self):
        self.layers = {}
        self.metadata = {}

    def Log(self, message: str) -> None:
        print(f"INFO: {message}")

    def Warnf(self, message: str, *args) -> None:
        print(f"WARN: {message}", args)

    def MkdirAll(self, path: str, mode: int) -> None:
        Path(path).mkdir(parents=True, exist_ok=True, mode=mode)

    def WriteFile(self, path: str, content: bytes, mode: int) -> None:
        with open(path, 'wb') as f:
            f.write(content)
            os.chmod(path, mode)

    def Exec(self, command: List[str], user_attribution: bool = False) -> None:
        subprocess.run(command, check=True)

@dataclass
class SyncRule:
    src: str
    dest: str

const_WATCHEXEC_LAYER = "watchexec"
const_WATCHEXEC_VERSION = "1.12.0"
const_WATCHEXEC_URL = "https://github.com/watchexec/watchexec/releases/download/{version}/watchexec-{version}-x86_64-unknown-linux-gnu.tar.xz"

class DevMode:
    def __init__(self, context: GCPContext):
        self.context = context

    @staticmethod
    def Enabled(context: GCPContext) -> bool:
        try:
            return os.environ["DEVMODE"] == "true"
        except KeyError as e:
            context.Warnf("Dev mode not enabled: %s", str(e))
            return False

    class Config:
        def __init__(self, build_cmd: List[str], run_cmd: List[str], ext: List[str]):
            self.build_cmd = build_cmd
            self.run_cmd = run_cmd
            self.ext = ext

    def add_file_watcher_process(self, config: Config) -> None:
        self.install_file_watcher()
        scripts_layer = self.context.layers.get("scripts", {"path": "scripts"})
        write_build_and_run_script(self.context, scripts_layer, config)
        self.context.AddWebProcess([const_WATCH_AND_RUN])

    def install_file_watcher(self) -> None:
        layer_name = const_WATCHEXEC_LAYER
        if layer_name not in self.context.layers:
            self.context.layers[layer_name] = {"path": layer_name}
        
        current_version = self.context.metadata.get(layer_name, {}).get("version", "")
        if current_version == const_WATCHEXEC_VERSION:
            self.context.Log(f"Using cached {layer_name}")
            return

        bin_dir = os.path.join(self.context.layers[layer_name]["path"], "bin")
        self.context.MkdirAll(bin_dir, 0o755)

        archive_url = const_WATCHEXEC_URL.format(version=const_WATCHEXEC_VERSION)
        command = f"curl --fail --show-error --silent --location --retry 3 {shlex.quote(archive_url)} | tar xJ --directory {shlex.quote(bin_dir)} --strip-components=1 --wildcards '*watchexec'"
        self.context.Exec(["bash", "-c", command])
        
        self.context.metadata[layer_name] = {"version": const_WATCHEXEC_VERSION}
