# Benchmark

## How to run benchmarks

**NON Production Servers only** Scripts are going to delete VMs and Templates. No questions asked.

Constraint: Your Proxmox must be able to give a VM on vmbr0 an IP via DHCP.

```bash
git clone https://github.com/egandro/proxmox-cpu-affinity
cd proxmox-cpu-affinity
scp -r ./benchmark proxmox:/benchmark
```

ssh to your proxmox

```bash
cd /benchmark

# customize
#cp /benchmark/env.template /benchmark/.env
#vi /benchmark/.env

# install dependencies
/benchmark/scripts/install-dependencies.sh

# install template
# dry-run
#/benchmark/scripts/orchestrator.py --create-templates --dry-run -v
/benchmark/scripts/orchestrator.py --create-templates

# show available testcases
/benchmark/scripts/orchestrator.py --show

# run testcase
# dry-run
/benchmark/scripts/orchestrator.py -t helloworld --dry-run -v
# run normal
/benchmark/scripts/orchestrator.py -t helloworld
# run verbose (print environment variables)
/benchmark/scripts/orchestrator.py -t helloworld -v
# run quiet (only the step names)
/benchmark/scripts/orchestrator.py -t helloworld -q
```

## Results

The results are stored in `/benchmark/results`.

A file `success` or `failed` indicated the status of the test.
