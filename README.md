# dirtop

`dirtop` lists per-directory CPU and memory usage of running processes.

Specify one or more directories — `dirtop` aggregates every process whose current working directory (`cwd`) is at or under each of them and prints them as a single row, side by side.

## Features

- Aggregate CPU% and RSS by directory, with PID count per row
- Compare multiple directories side by side in one table
- Expand a parent directory into its subdirectories (`--depth N`)
- Nested top-N processes per directory (`--top-procs`) with cwd hint
- Full command line view (`--full-cmd`)
- JSON output (`--json`) and continuously refreshing TUI (`--watch`)
- macOS and Linux

## Install

**homebrew tap:**

```console
$ brew install k1LoW/tap/dirtop
```

**manually:**

Download a binary from the [releases page](https://github.com/k1LoW/dirtop/releases).

## Usage

```console
$ dirtop ~/src/github.com/k1LoW/dirtop ~/src/github.com/k1LoW/dirmap
DIR                                       PID(S)  COMMAND  CPU%  MEM(RSS)
/Users/k1low/src/github.com/k1LoW/dirtop  (7)              18.9  163.2MiB
/Users/k1low/src/github.com/k1LoW/dirmap  (0)              0.0   0B
```

Expand a parent directory into its immediate subdirectories (one row each):

```console
$ dirtop --depth 1 --sort cpu ~/src/github.com/k1LoW
DIR                                           PID(S)  COMMAND  CPU%  MEM(RSS)
/Users/k1low/src/github.com/k1LoW/dirtop      (8)              24.0  253.0MiB
/Users/k1low/src/github.com/k1LoW/animNumber  (5)              7.4   84.5MiB
/Users/k1low/src/github.com/k1LoW/kakitori    (9)              6.0   110.7MiB
...
```

Show the top processes inside each directory:

```console
$ dirtop --top-procs 3 ~/src/github.com/k1LoW
DIR                                PID(S)  COMMAND           CPU%  MEM(RSS)
/Users/k1low/src/github.com/k1LoW  (28)                      32.5  513.8MiB
├─                                  3386   claude (dirtop/)  26.6  204.5MiB
├─                                 55302   dirtop (dirtop/)  1.7   10.2MiB
└─                                 88138   claude (animJ/)   1.7   33.9MiB
```

Watch mode (Ctrl-C / `q` to quit):

```console
$ dirtop --watch --top-procs 5 ~/src/foo ~/src/bar
```

JSON output:

```console
$ dirtop --json --top-procs 3 ~/src/foo | jq .
```

### Behavior notes

- Processes whose `cwd` cannot be read (typically those owned by other users) are silently dropped.
- Directories with no processes (`PIDS=0`) are hidden by default. Pass `--show-empty` to include them.
- When two arguments are in a parent/child relationship, each process is counted toward the longest-prefix match only — no double counting.
- CPU% is the gopsutil raw value, summed across cores (i.e. can exceed 100% on multi-core systems).

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--depth` | `0` | Expand each DIR into subdirectories exactly N levels below (`0` = off) |
| `--top-procs` | `0` | Show top N processes per directory (`0` = off) |
| `--sort` | `input` | Sort rows by `input` / `cpu` / `mem` / `pids` |
| `--full-cmd` | `false` | Show full command line in nested rows |
| `--show-empty` | `false` | Include rows with `PIDS=0` |
| `--json` | `false` | JSON output |
| `--watch`, `-w` | `false` | Continuously refresh (TUI) |
| `--interval` | `500ms` | CPU sampling interval |

## Build

```console
$ make build
```

## License

[MIT License](LICENSE)
