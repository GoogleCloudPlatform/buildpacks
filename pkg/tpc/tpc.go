import os
from pkg.env import TPCTarballProject, TPCHostname

def is_tpc() -> bool:
    """
    Returns true if the build universe is set and is not GDU.
    """
    _, present = get_tarball_project()
    return present

def get_tarball_project() -> tuple[str, bool]:
    """
    Returns the Artifact Registry project for the TPC tarball.
    """
    project = os.getenv(TPCTarballProject)
    if project:
        return project, True
    return "", False

def get_hostname() -> tuple[str, bool]:
    """
    Returns the hostname for the TPC build.
    """
    hostname = os.getenv(TPCHostname)
    if hostname:
        return hostname, True
    return "", False
