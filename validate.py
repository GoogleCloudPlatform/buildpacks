import subprocess
import os
import uuid 
import time
import sys
import argparse

def run_pack_build_and_docker_run(target_image, app_dir, builder_image, runtime_version):
    """
    Runs the 'pack build' command to create a Docker image of your Cloud Function,
    and then runs the generated Docker image as a container.

    Args:
        target_image (str): The desired name for the Docker image to be built and run.
        app_dir (str): The path to the directory containing your Cloud Function's source code.
        builder_image (str): The name of the Cloud Native Buildpacks builder image to use
        (e.g., for Python functions: 'gcr.io/buildpacks/google-cloud-functions/python311:latest').
        function_target (str): The name of the entry-point function within your code
                                (e.g., 'hello_world' if your function is `def hello_world(request):`).
        runtime_version (str): The specific runtime version of your function
        (e.g., '3.11' for Python 3.11).
    """

    # --- 1. Run pack build command ---
    print(f"--- 🚀 Starting Pack Build for image: {target_image} ---")
    pack_command_args = [
        "pack",
        "build",
        target_image,
        "--path", app_dir,
        "--builder", builder_image,
        "--env", f"GOOGLE_RUNTIME_VERSION={runtime_version}"
    ]

    try:
        pack_result = subprocess.run(
            pack_command_args,
            capture_output=True,
            text=True,
            check=True
        )
        print("Pack build command executed successfully! 🎉")
        print("\n--- Pack Build Standard Output ---")
        print(pack_result.stdout)
        if pack_result.stderr:
            print("\n--- Pack Build Standard Error ---")
            print(pack_result.stderr)

    except FileNotFoundError:
        print(f"\nError: 'pack' command not found. 🚨")
        print("Please ensure Cloud Native Buildpacks 'pack' CLI is installed and in your system's PATH.")
        print("Download: https://buildpacks.io/docs/install-pack/")
        return # Exit if pack is not found
    except subprocess.CalledProcessError as e:
        print(f"\nError executing pack build command. Exit code: {e.returncode} ❌")
        print("This usually means there was an issue building your application image.")
        print("\n--- Pack Build Standard Output ---")
        print(e.stdout)
        print("\n--- Pack Build Standard Error ---")
        print(e.stderr)
        return # Exit if pack build fails
    except Exception as e:
        print(f"\nAn unexpected error occurred during pack build: {e} 🐛")
        return # Exit for other unexpected errors

    print("\n" + "="*70 + "\n") # Separator

    # --- 2. Run docker run command ---
    print(f"--- 🐳 Attempting to run image: {target_image} with Docker ---")
    docker_command_args = [
        "docker",
        "run",
        "-d",
        # "--rm",         # Automatically remove the container when it exits
        # "-p", "8080:8080", # Map host port 8080 to container port 8080 (standard for Cloud Functions)
        target_image
    ]

    container_id = None # Initialize container_id outside try block

    try:
        docker_run_result = subprocess.run(
            docker_command_args,
            capture_output=True,
            text=True,
            check=True # Raise an exception if docker run fails to start
        )
        
        container_id = docker_run_result.stdout.strip()
        # print(f"Docker container '{container_id}' for '{target_image}' started successfully. 🎉")
        # print(f"Your function should be accessible at http://localhost:8080")
        # print(f"Checking container status for 5 seconds... ⏳")

        # --- 3. Check container status (simplified health check) ---
        # We'll check if the container is still running after a short delay
        # For more robust health checks, you'd query specific /health endpoints or logs.
        time.sleep(3) # Give the container a moment to initialize and potentially crash

        status_result = subprocess.run(
            ["docker", "inspect", "-f", "{{.State.Status}}", container_id],
            capture_output=True,
            text=True,
            check=True
        )
        container_status = status_result.stdout.strip()

        if container_status == "running":
            print(f"\nDocker container '{container_id}' is running as expected! ✅")
            print("It's fine. Proceeding to stop the container gracefully. 👍")
            
            # --- 4. Stop the container ---
            print(f"Stopping container '{container_id}'... 🛑")
            subprocess.run(["docker", "stop", container_id], check=True)
            print(f"Docker container '{container_id}' stopped successfully. ✅")
            sys.exit(0) # Exit with success
        else:
            print(f"\nError: Docker container '{container_id}' is not running (status: {container_status}) ❌")
            print("This indicates the container failed shortly after starting.")
            print(f"To view logs for debugging: docker logs {container_id}")
            sys.exit(1) # Exit with error

    except FileNotFoundError:
        print(f"\nError: 'docker' command not found. 🚨")
        print("Please ensure Docker Desktop/Engine is installed and in your system's PATH.")
        print("Download Docker: https://www.docker.com/get-started/")
    except Exception as e:
        print(f"\nAn unexpected error occurred during docker run: {e} 🐛")


if __name__ == "__main__":

    parser = argparse.ArgumentParser(description="Build and run a Google Cloud Function using Pack and Docker.")
    parser.add_argument("--app-dir", required=True, help="Path to the application directory containing your source code.")
    parser.add_argument("--version", required=True, help="Version of the Cloud Function runtime (e.g., '3.11' for Python, '18' for Node.js).") # New argument

    args = parser.parse_args()
    # --- Configuration for your Cloud Function ---
    # !! IMPORTANT: Update these variables for your specific function !!

    # The name for the Docker image that 'pack build' will create.
    # For local testing, a simple name like "my-local-function" is fine.
    # For deploying to Google Cloud Registry, it would be "gcr.io/your-project-id/your-function-name"
    random_id = str(uuid.uuid4())[:8]
    my_target_image = f"{args.app_dir}-{args.version}-app-{random_id}"

    # The path to the directory containing your Cloud Function's source code.
    # E.g., if your function is in a folder named 'my_awesome_function' next to this script, use './my_awesome_function'.
    base_app_dir= "./builders/testdata/"

    my_app_dir = os.path.join(base_app_dir, args.app_dir)

    # The Cloud Native Buildpacks builder image for your function's runtime.
    # Common examples:
    # Python: "gcr.io/buildpacks/google-cloud-functions/python311:latest"
    # Node.js: "gcr.io/buildpacks/google-cloud-functions/nodejs18:latest"
    # Go: "gcr.io/buildpacks/google-cloud-functions/go121:latest"
    my_builder_image = "gcr.io/buildpacks/builder:latest"

    # The name of the entry-point function within your code.
    # If your Python code has `def my_entry_point(request):`, then this should be "my_entry_point".

    # The runtime version. This should match the builder image's runtime.
    # Python: "3.11"
    # Node.js: "18"
    # Go: "1.21"
    my_runtime_version = args.version

    print(f"Checking for app directory at: {my_app_dir}")
    if not os.path.exists(my_app_dir):
        os.makedirs(my_app_dir)
        print(f"Created dummy app directory: {my_app_dir}")

    print("\n" + "~"*70 + "\n") # Separator

    # --- Execute the main function ---
    run_pack_build_and_docker_run(
        my_target_image,
        my_app_dir,
        my_builder_image,
        my_runtime_version
    )
