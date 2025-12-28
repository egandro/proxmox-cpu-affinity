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

def execute_scripts(scripts, env, context_name, args):
    """Iterates over scripts and executes them."""
    for step_name, step_config in scripts.items():
        # Merge Step-specific Env
        step_env = env.copy()
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
            print(f"Aborting '{context_name}' due to failure in step '{step_name}'.")
            return False
    return True

def main():
    parser = argparse.ArgumentParser(description="Benchmark Orchestrator")
    parser.add_argument("config", nargs="?", default="/benchmark/testcases.json", help="Path to configuration file")
    parser.add_argument("--dry-run", action="store_true", help="Simulate execution")
    parser.add_argument("-v", "--verbose", action="store_true", help="Enable verbose output")
    parser.add_argument("-q", "--quiet", action="store_true", help="Suppress command output")
    parser.add_argument("-t", "--testcase", help="Only run the specified testcase")
    parser.add_argument("--create-templates", action="store_true", help="Run template creation scripts")
    args = parser.parse_args()

    if not os.path.exists(args.config):
        print(f"Error: {args.config} not found.")
        sys.exit(1)

    with open(args.config, 'r') as f:
        data = json.load(f)

    global_env = data.get("global_config", {}).get("env", {})

    # Determine execution targets
    if args.create_templates:
        if "create_templates" not in data:
            print("Warning: 'create_templates' section not found in configuration.")
            targets = []
        else:
            targets = [data["create_templates"]]
    else:
        targets = data.get("testcases", [])

    for testcase in targets:
        if "name" in testcase:
            context_name = testcase["name"]
        elif args.create_templates:
            context_name = "create_templates"
        else:
            print("Error: Testcase definition missing required 'name' field.")
            sys.exit(1)

        if not args.create_templates and args.testcase and args.testcase != context_name:
            continue

        print(f"\n========================================")
        print(f"RUNNING: {context_name}")
        print(f"========================================")

        # Setup Environment
        tc_env = global_env.copy()
        if not args.create_templates:
            tc_env["TESTCASE"] = context_name
        tc_env.update(testcase.get("env", {}))

        scripts = testcase.get("scripts", {})
        if not execute_scripts(scripts, tc_env, context_name, args):
            if args.create_templates:
                sys.exit(1)

if __name__ == "__main__":
    main()