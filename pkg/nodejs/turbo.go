import pkg.nodejs.gcpbuildpack as gcp
from typing import List

vulnerable_nextjs_ranges = [
    ">= 15.0.0-0, < 15.0.5",
    ">= 15.1.0-0, < 15.1.9",
    ">= 15.2.0-0, < 15.2.6",
    ">= 15.3.0-0, < 15.3.6",
    ">= 15.4.0-0, < 15.4.8",
    ">= 15.5.0-0, < 15.5.7",
    ">= 16.0.0-0, < 16.0.7",
    ">= 15.6.0-canary.0, < 15.6.0-canary.58",
    ">= 16.1.0-canary.0, < 16.1.0-canary.12",
    ">= 14.3.0-canary.77, < 15.0.0",
]

vulnerable_rsc_ranges = [
    ">= 19.0.0-0, < 19.0.1",
    ">= 19.1.0-0, < 19.1.2",
    ">= 19.2.0-0, < 19.2.1",
]

target_react_server_packages = [
    "react-server-dom-webpack",
    "react-server-dom-parcel",
    "react-server-dom-turbopack",
]

def check_vulnerabilities(ctx: gcp.Context, node_deps: 'NodeDependencies') -> str:
    if os.environ.get(env.ALLOW_VULNERABLE_DEPENDENCIES, "").lower() == "true":
        ctx.warn(f"Skipping vulnerability checks because {env.ALLOW_VULNERABLE_DEPENDENCIES} is enabled.")
        return ""
    
    next_version = get_version(node_deps, "next")
    if next_version:
        for range in vulnerable_nextjs_ranges:
            matches, err = version_matches_semver(ctx, range, next_version)
            if err:
                ctx.warn(f"could not check for vulnerable Next.js version: {err}")
                continue
            if matches:
                return gcp.user_error(
                    f"vulnerable Next.js version {next_version} detected due to CVE-2025-55182. "
                    "Please upgrade to a patched version (e.g., 15.0.5, 15.1.9, 15.5.7, 16.0.7). "
                    "See https://github.com/vercel/next.js/security/advisories/GHSA-9qr9-h5gf-34mp for details."
                )
    
    for pkg_name in target_react_server_packages:
        rsc_version = get_version(node_deps, pkg_name)
        if not rsc_version:
            continue
        
        for range in vulnerable_rsc_ranges:
            matches, err = version_matches_semver(ctx, range, rsc_version)
            if err:
                ctx.warn(f"could not check for vulnerable {pkg_name} version: {err}")
                continue
            if matches:
                return gcp.user_error(
                    f"vulnerable {pkg_name} version {rsc_version} detected due to CVE-2025-55182. "
                    "Please upgrade to a patched version (e.g., 19.0.1, 19.1.2, 19.2.1). "
                    "See https://react.dev/blog/2025/12/03/critical-security-vulnerability-in-react-server-components for details."
                )
    
    return ""
