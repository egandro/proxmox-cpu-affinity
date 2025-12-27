#!/usr/bin/env python3
import json
import subprocess
import os
import sys
import argparse

def run_command(command, env_vars, step_name, dry_run=False, verbose=False, quiet=False):
    """Executes a shell command with the specific environment."""
    display_cmd = command
    prefix = "[EXEC]"

    if dry_run:
        prefix = "[DRY-RUN]"
        if verbose:
            # Expand the variables
            prefix = "[EXPANDED]"
            for key in sorted(env_vars.keys(), key=len, reverse=True):
                val = env_vars[key]
                display_cmd = display_cmd.replace(f"${key}", val).replace(f"${{{key}}}", val)

    print(f"  {prefix} {step_name}: {display_cmd}")

    if verbose and len(env_vars) > 0:
        for key, val in env_vars.items():
            print(f"      {key}={val}")

    if dry_run:
        return

    try:
        # Merge system env with our custom env
        full_env = os.environ.copy()
        full_env.update(env_vars)

        stdout_dest = None
        stderr_dest = None
        if quiet:
            stdout_dest = subprocess.DEVNULL
            stderr_dest = subprocess.DEVNULL

        subprocess.run(
            command,
            shell=True,
            env=full_env,
            check=True,
            executable='/bin/bash',
            stdout=stdout_dest,
            stderr=stderr_dest
        )
    except subprocess.CalledProcessError as e:
        print(f"  [ERROR] Step '{step_name}' failed with exit code {e.returncode}")
        raise e

def main():
    parser = argparse.ArgumentParser(description="Benchmark Orchestrator")
    parser.add_argument("config", nargs="?", default="/benchmark/testcases.json", help="Path to configuration file")
    parser.add_argument("--dry-run", action="store_true", help="Simulate execution")
    parser.add_argument("-v", "--verbose", action="store_true", help="Enable verbose output")
    parser.add_argument("-q", "--quiet", action="store_true", help="Suppress command output")
    parser.add_argument("-t", "--testcase", help="Only run the specified testcase")
    args = parser.parse_args()

    if not os.path.exists(args.config):
        print(f"Error: {args.config} not found.")
        sys.exit(1)

    with open(args.config, 'r') as f:
        data = json.load(f)

    global_env = data.get("global_config", {}).get("env", {})

    for testcase in data.get("testcases", []):
        if "name" not in testcase:
            print("Error: Testcase definition missing required 'name' field.")
            sys.exit(1)

        tc_name = testcase["name"]
        if args.testcase and args.testcase != tc_name:
            continue

        print(f"\n========================================")
        print(f"RUNNING TESTCASE: {tc_name}")
        print(f"========================================")

        # 1. Setup Base Environment (Global + Testcase)
        tc_env = global_env.copy()
        tc_env["TESTCASE"] = tc_name
        tc_env.update(testcase.get("env", {}))

        scripts = testcase.get("scripts", {})

        # 2. Iterate EXACTLY as defined in the JSON file
        for step_name, step_config in scripts.items():
            # Merge Step-specific Env
            step_env = tc_env.copy()
            step_env.update(step_config.get("env", {}))

            # Handle "cmds" (list) vs "cmd" (string)
            commands = []
            if "cmds" in step_config:
                commands = step_config["cmds"]
            elif "cmd" in step_config:
                commands = [step_config["cmd"]]

            # Execute
            try:
                for cmd in commands:
                    run_command(cmd, step_env, step_name, dry_run=args.dry_run, verbose=args.verbose, quiet=args.quiet)
            except subprocess.CalledProcessError:
                print(f"Aborting testcase '{tc_name}' due to failure in step '{step_name}'.")
                break

if __name__ == "__main__":
    main()