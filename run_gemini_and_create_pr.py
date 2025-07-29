
import os
import subprocess
import argparse
import requests
import json

def run_gemini_cli(prompt):
    """Runs the Gemini CLI with the given prompt."""
    try:
        print(f"Running Gemini CLI with prompt: {prompt}")
        # Replace 'gemini' with the actual command if it's different
        subprocess.run(['gemini', '--yolo'], input=prompt.encode(), check=True)
        print("Gemini CLI executed successfully.")
    except FileNotFoundError:
        print("Error: 'gemini' command not found. Make sure the Gemini CLI is installed and in your PATH.")
        exit(1)
    except subprocess.CalledProcessError as e:
        print(f"Error executing Gemini CLI: {e}")
        exit(1)

def create_github_pr(repo_url, head_branch, base_branch, title, body):
    """Creates a GitHub Pull Request."""
    github_token = os.environ.get("GITHUB_TOKEN")
    if not github_token:
        print("Error: GITHUB_TOKEN environment variable not set.")
        print("Please set it to your GitHub Personal Access Token with 'repo' scope.")
        exit(1)

    api_url = f"{repo_url.replace('github.com', 'api.github.com/repos')}/pulls"
    headers = {
        "Authorization": f"token {github_token}",
        "Accept": "application/vnd.github.v3+json",
    }
    data = {
        "title": title,
        "body": body,
        "head": head_branch,
        "base": base_branch,
    }

    print(f"Creating Pull Request to {repo_url} from '{head_branch}' to '{base_branch}'")
    response = requests.post(api_url, headers=headers, data=json.dumps(data))

    if response.status_code == 201:
        print("Pull Request created successfully:")
        print(response.json()["html_url"])
    else:
        print(f"Error creating Pull Request: {response.status_code}")
        print(response.json())

def main():
    parser = argparse.ArgumentParser(description="Run Gemini CLI and create a GitHub PR.")
    parser.add_argument("prompt", help="The prompt to pass to the Gemini CLI.")
    parser.add_argument("--repo", default="https://github.com/maigovannon/buildpacks-abhinasu", help="The GitHub repository URL.")
    parser.add_argument("--branch", default="main", help="The base branch for the pull request.")
    parser.add_argument("--title", help="The title of the pull request. Defaults to the prompt.")
    parser.add_argument("--body", default="PR created by Gemini CLI automation script.", help="The body of the pull request.")
    args = parser.parse_args()

    pr_title = args.title if args.title else args.prompt
    
    # 1. Create a new branch for the changes
    new_branch = f"gemini-changes-{os.urandom(4).hex()}"
    print(f"Creating and switching to new branch: {new_branch}")
    subprocess.run(['git', 'checkout', '-b', new_branch], check=True)

    # 2. Run the Gemini CLI
    run_gemini_cli(args.prompt)

    # 3. Stage and commit the changes
    print("Staging changes...")
    subprocess.run(['git', 'add', '.'], check=True)
    
    print("Committing changes...")
    subprocess.run(['git', 'commit', '-m', pr_title], check=True)

    # 4. Push the new branch
    print(f"Pushing branch {new_branch} to origin...")
    subprocess.run(['git', 'push', '-u', 'origin', new_branch], check=True)

    # 5. Create the Pull Request
    create_github_pr(args.repo, new_branch, args.branch, pr_title, args.body)

if __name__ == "__main__":
    main()

