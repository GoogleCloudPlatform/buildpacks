import json
from pathlib import Path
from typing import Optional

class NodeJS:
    @staticmethod
    def nuxt_start_command(application_root: str) -> Optional[list[str]]:
        config_path = Path(application_root) / "nuxt.config.ts"
        server_path = Path(application_root) / ".output/server/index.mjs"
        
        if config_path.exists() and server_path.exists():
            return ["node", ".output/server/index.mjs"]
        return None
