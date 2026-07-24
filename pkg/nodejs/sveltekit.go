class NodeJS:
    @staticmethod
    def detect_svelte_kit_auto_adapter(package_json: dict) -> bool:
        dev_deps = package_json.get("devDependencies", {})
        
        # Check for adapter-auto and no other adapters
        has_auto = False
        for dep in dev_deps.keys():
            if dep.startswith("@sveltejs/adapter-"):
                if dep == "@sveltejs/adapter-auto":
                    has_auto = True
                else:
                    return False
        
        return has_auto

    @staticmethod
    def svelte_kit_start_command(application_root: str) -> Optional[list[str]]:
        config_path = Path(application_root) / "svelte.config.js"
        server_path = Path(application_root) / "build/index.js"
        
        if config_path.exists() and server_path.exists():
            return ["node", "build/index.js"]
        return None
