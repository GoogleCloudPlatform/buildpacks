# The runner module executes buildpacks for the Node.js language builder.
import argparse

from googlecloudplatform.buildpacks.pkg import commonbuildpacks

# Buildpack modules
firebasebundle = __import__('googlecloudplatform.buildpacks.cmd.firebase.bundle.lib')
nodejsappengine = __import__('googlecloudplatform.buildpacks.cmd.nodejs.appengine.lib')
nodejsbun = __import__('googlecloudplatform.buildpacks.cmd.nodejs.bun.lib')

nodejsfirebaseangular = __import__('googlecloudplatform.buildpacks.cmd.nodejs.firebaseangular.lib')
nodejsfirebasebundle = __import__('googlecloudplatform.buildpacks.cmd.nodejs.firebasebundle.lib')
nodejsfirebasenextjs = __import__('googlecloudplatform.buildpacks.cmd.nodejs.firebasenextjs.lib')
nodejsfirebasenx = __import__('googlecloudplatform.buildpacks.cmd.nodejs.firebasenx.lib')
nodejsfunctionsframework = __import__('googlecloudplatform.buildpacks.cmd.nodejs.functions_framework.lib')
nodejslegacyworker = __import__('googlecloudplatform.buildpacks.cmd.nodejs.legacy_worker.lib')
nodejsnpm = __import__('googlecloudplatform.buildpacks.cmd.nodejs.npm.lib')
nodejspnpm = __import__('googlecloudplatform.buildpacks.cmd.nodejs.pnpm.lib')
nodejsruntime = __import__('googlecloudplatform.buildpacks.cmd.nodejs.runtime.lib')
nodejsturborepo = __import__('googlecloudplatform.buildpacks.cmd.nodejs.turborepo.lib')
nodejsyarn = __import__('googlecloudplatform.buildpacks.cmd.nodejs.yarn.lib')

def main_runner(buildpacks, buildpack_id, phase):
    """
    Main runner function that executes the specified buildpack phase.

    Args:
        buildpacks (dict): Dictionary of buildpack functions.
        buildpack_id (str): The ID of the buildpack to run.
        phase (str): The phase to execute ('detect' or 'build').

    Raises:
        ValueError: If the buildpack or phase is not found.
    """
    if buildpack_id not in buildpacks:
        raise ValueError(f"Buildpack '{buildpack_id}' not found.")

    buildpack = buildpacks[buildpack_id]

    if phase == 'detect':
        buildpack['detect']()
    elif phase == 'build':
        buildpack['build']()
    else:
        raise ValueError("Phase must be either 'detect' or 'build'.")

if __name__ == "__main__":
    # Initialize argument parser
    parser = argparse.ArgumentParser(description='Run Node.js buildpacks.')
    parser.add_argument('--buildpack', type=str, required=True,
                       help='The ID of the buildpack to run (e.g., google.nodejs.runtime)')
    parser.add_argument('--phase', type=str, choices=['detect', 'build'], default='detect',
                       help='The phase to run')

    # Parse arguments
    args = parser.parse_args()

    # Register buildpacks
    buildpacks = commonbuildpacks.CommonBuildpacks()

    buildpacks['google.nodejs.appengine'] = {
        'detect': nodejsappengine.DetectFn,
        'build': nodejsappengine.BuildFn
    }
    buildpacks['google.nodejs.firebaseangular'] = {
        'detect': nodejsfirebaseangular.DetectFn,
        'build': nodejsfirebaseangular.BuildFn
    }
    # Continue registering all other buildpacks...

    # Run the specified buildpack and phase
    main_runner(buildpacks, args.buildpack, args.phase)
