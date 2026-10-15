# LimaVM TUI (`limavm-tui`)

> Interactive Terminal UI (TUI) for managing [Lima](https://github.com/lima-vm/lima) (`limactl`) virtual machines.

Built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [Bubbles](https://github.com/charmbracelet/bubbles).

Like `k9s` or `lazydocker`, `limavm-tui` is a single static binary, single process, keyboard-driven interface with live-updating panels, interactive shell passthrough, and zero external database dependencies.

---

## Features

- **Dashboard (VM List)**: Real-time table of Lima instances with CPU, Memory, Disk, IP, Arch, and status indicators.
- **Full VM Lifecycle**: Start, stop, restart, delete (with confirmation dialog), protect/unprotect, rename (emulated via clone + delete), and clone.
- **Embedded Interactive Shell**: Press `s` on any running VM to suspend the TUI and drop directly into `limactl shell <name>` with full terminal PTY passthrough, resuming the TUI automatically upon exit.
- **VM Detail View**: Tabbed inspection of VM metadata:
  - **Overview**: Hardware specs, directories, Lima version, and error logs.
  - **Disks**: Attached and global virtual disks with create, delete, resize, and unlock keybindings.
  - **Network & Tunnels**: Host interfaces, attached MACs, SSH config snippets, and instant SOCKS tunnel launching (`limactl tunnel`).
  - **Snapshots**: Point-in-time snapshot list with create, apply (restore), and delete keybindings.
  - **YAML**: Live view of `lima.yaml` with one-key edit in `$EDITOR` (`e`) and automatic validation.
- **VM Creation Wizard**: Template catalog with live description preview, or custom YAML path/URL input with override fields for CPU, Memory, and Disk.
- **Networks Screen**: Dedicated management of virtual networks (`limactl network create/delete`).
- **VM Profiles**: Drive an external Lima harness (its own template, scripts and provisioning steps) straight from the TUI, declared in a small YAML profile file that the harness repository ships itself. See [Profiles](#profiles).
- **Prune & Diagnostics**: Maintenance actions behind confirmation dialogs (`P` to prune unused objects, `I` for diagnostic host info).
- **Persistent Local Preferences**: Favorites and last-used template saved in a small local JSON file (`~/.local/share/limavm-tui/state.json`).

---

## Prerequisites

1. **Lima (`limactl`)**: Version 1.0.0 or later (verified on `limactl 1.2.3`).
   ```bash
   limactl --version
   ```
2. **VM Driver**: A supported hypervisor backend configured on the host:
   - macOS: `vz` (Virtualization.framework) or `qemu`
   - Linux: `qemu` / KVM
3. **Go**: Version 1.22+ (for building from source).

---

## Installation

### From Source (Makefile)

```bash
git clone https://github.com/masterivanic/limaVM-TUI.git
cd limaVM-TUI


# Build static binary to ./bin/limavm-tui
make build

# Or install directly to $GOPATH/bin
make install
```

### Running

```bash
# Launch TUI
./bin/limavm-tui

# Or with custom options
limavm-tui --refresh 2s --data-dir ~/.config/limavm-tui --log /tmp/limavm.log
```

---

## Command Line Options

| Flag | Default | Description |
|---|---|---|
| `--refresh` | `3s` | Polling interval for instance and network live status |
| `--data-dir` | `~/.local/share/limavm-tui` | Directory storing persistent state (favorites, last-used template) |
| `--profile` | — | Path to a VM profile file, or a directory containing `lima-ui.profile.yaml`. Repeatable. See [Profiles](#profiles) |
| `--config` | `~/.config/limavm-tui/config.yaml` | Config file listing profile search paths |
| `--log` | `""` | Optional debug log file path |
| `--version` | `false` | Print version and build metadata |
| `-h`, `--help`| — | Show help information |

---

## Keyboard Shortcuts Cheat Sheet

Press `?` inside the application at any time to open the help overlay.

### Global Navigation

| Key | Action |
|---|---|
| `1` | Jump to **Dashboard** (VM list) |
| `2` | Jump to **Networks** screen |
| `3` | Jump to **Create VM** form |
| `4` | Jump to **Profiles** screen |
| `I` | Open **Host Diagnostics & Info** modal |
| `P` | **Prune** unused Lima objects (with confirmation) |
| `?` | Toggle **Help Overlay** |
| `q` / `Esc` | Return to Dashboard / Quit application |
| `Ctrl+C` | Force quit |

### Dashboard (VM List)

| Key | Action |
|---|---|
| `Enter` | Open **VM Details** |
| `s` | Open interactive **Shell** in VM (`limactl shell`) |
| `x` | **Start / Stop** VM (toggle) |
| `d` | **Delete** VM (opens confirmation dialog) |
| `c` | **Create** new VM |
| `r` | **Rename** VM (clones to new name, deletes original) |
| `C` | **Clone** VM |
| `p` | **Protect / Unprotect** VM against deletion |
| `f` | Toggle **Favorite** (stars instance at top) |
| `/` | **Filter** VM list by name, status, or arch |
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |

### VM Detail Screen

| Key | Action |
|---|---|
| `Tab` / `Right` / `l` | Next tab |
| `Shift+Tab` / `Left` / `h` | Previous tab |
| `1` - `5` | Direct jump to tab (Overview, Disks, Network, Snapshots, YAML) |
| `s` | Open interactive Shell in this VM |
| `t` | Launch SOCKS tunnel (`limactl tunnel`) |
| `e` | Edit `lima.yaml` in `$EDITOR` (YAML tab) |
| `Esc` / `q` | Return to Dashboard |

### Disks Tab (`2`)

| Key | Action |
|---|---|
| `c` | Create new virtual disk (`limactl disk create`) |
| `r` | Resize selected disk (`limactl disk resize`) |
| `u` | Unlock selected disk (`limactl disk unlock`) |
| `d` | Delete selected disk (with confirmation) |
| `j` / `k` | Select disk |

### Snapshots Tab (`4`)

| Key | Action |
|---|---|
| `c` | Take a new snapshot (`limactl snapshot create`) |
| `a` | Apply / restore snapshot (`limactl snapshot apply`) |
| `d` | Delete snapshot (`limactl snapshot delete`) |
| `j` / `k` | Select snapshot |

### Networks Screen (`2`)

| Key | Action |
|---|---|
| `c` | Create virtual network (`limactl network create`) |
| `d` | Delete network (`limactl network delete -f`) |
| `r` | Refresh network table |
| `1` / `Esc` | Return to Dashboard |

### Profiles Screen (`4`)

| Key | Action |
|---|---|
| `↑` / `↓` | Select a profile (left pane) or a provisioning mode (right pane) |
| `Tab` | Switch focus between profiles and provisioning modes |
| `u` | Start the VM declared by the profile (`lifecycle.create`) |
| `h` | Run the profile health check (`lifecycle.healthCheck`) |
| `p` / `Enter` | Run the selected provisioning mode |
| `Esc` | Return to Dashboard |

---

## Profiles

A **profile** lets `limavm-tui` drive an existing Lima *harness* — a repository that
already owns a Lima template, some provisioning scripts and a `make`-based workflow —
without that knowledge being baked into this tool.

The harness repository ships a single YAML file (conventionally `lima-ui.profile.yaml`
at its root) describing where its template and scripts are and what they do. Nothing is
copied or vendored: `limavm-tui` only needs the path to that file.

```sh
# Point at the profile file, or at the directory containing it
limavm-tui --profile ~/sources/pro/rdb-dev/lima-ui.profile.yaml
limavm-tui --profile ~/sources/pro/rdb-dev

# Repeat the flag to load several harnesses at once
limavm-tui --profile ~/sources/pro/rdb-dev --profile ~/sources/pro/other-harness
```

To avoid retyping the flag, list search paths in `~/.config/limavm-tui/config.yaml`:

```yaml
profilePaths:
  - ~/sources/pro/rdb-dev
  - ~/sources/pro/other-harness/lima-ui.profile.yaml
```

Then press **`4`** in the TUI to open the Profiles screen: pick a profile, start its VM
(`u`), run its health check (`h`) and run one of its provisioning modes (`p`). Each
action runs the harness' own script with the terminal handed over to it, so its output
is streamed live exactly as if it had been launched from a shell.

### Profile schema

```yaml
apiVersion: v1                        # required, currently always "v1"
name: rdb-dev                         # required, shown in the profile list
description: Rocky 9 VM harness for testing RDB products
vmName: ansible-rocky9                # required, the Lima instance name

template:
  source: lima/ansible-rocky9.yaml.in # the harness' Lima template
  output: vm_config/ansible-rocky9.yaml  # where the rendered YAML is produced
  render:                             # optional custom rendering step
    script: scripts/render-lima-template.sh
    args: ["${TEMPLATE_OUT}", "${VM_TYPE}", "..."]

defaults:                             # exposed to commands as placeholders
  vmType: qemu
  mountType: reverse-sshfs
  arch: x86_64
  rosetta: "false"
  cpus: 4
  memory: 6G
  disk: 40G
  configDir: config
  workspaceDir: workspace
  portForward: ""

mounts:                               # documented for display
  - host: config
    guest: /opt/ansible-test/provisioning

lifecycle:
  create:                             # brings the VM up
    script: scripts/vm-up.sh
    args: ["${VM_NAME}", "${TEMPLATE_OUT}", "..."]
  healthCheck:                        # verifies the VM is usable
    script: scripts/vm-check.sh       # args default to ["${VM_NAME}"]

provisioningModes:                    # named, ordered sequences of steps
  - name: standalone-rdb
    description: Install RPMs and load podman images
    steps:
      - script: scripts/vm-install-standalone-rpms.sh
      - script: scripts/vm-load-podman-images.sh
```

`lifecycle.create` is an escape hatch: a harness that already owns its bring-up logic
(rendering, port forwarding, post-creation fixups) declares its own script here, and it
is delegated to as-is rather than reimplemented — so no behaviour is lost or duplicated.

### Placeholders

Command arguments may reference these, expanded from `vmName` and `defaults`:

`${VM_NAME}` · `${PROFILE_NAME}` · `${PROFILE_ROOT}` · `${VM_TYPE}` · `${MOUNT_TYPE}` ·
`${ARCH}` · `${ROSETTA}` · `${CPUS}` · `${MEMORY}` · `${DISK}` · `${CONFIG_DIR}` ·
`${WORKSPACE_DIR}` · `${PORT_FORWARD}` · `${TEMPLATE_OUT}` · `${TEMPLATE_SRC}`

When a command declares no `args`, it defaults to `["${VM_NAME}"]`, the calling
convention of most harness scripts.

### Execution model

- Scripts run with the **profile directory as working directory**, so a harness can call
  its sibling scripts and resolve its assets through relative paths as usual.
- Arguments are passed as **argv, never through a shell**, so a profile cannot inject
  shell syntax and values containing spaces stay a single argument.
- Every path a profile references **must stay inside the profile directory**, must exist
  and (for scripts) must be executable. A profile causes external programs to run, so it
  is not allowed to point outside its own repository.
- Provisioning steps run **one at a time**; a failing step aborts the remaining sequence.
- A profile that fails to load is **reported**, in the terminal at startup and in the
  Profiles screen, rather than silently ignored.

---

## To Run vm terminal in web interface

1. Install tty

```bash
limactl shell instane-name -- sudo curl -L \
  https://github.com/tsl0922/ttyd/releases/latest/download/ttyd.x86_64 \
  -o /usr/local/bin/ttyd
limactl shell instane-name -- sudo chmod +x /usr/local/bin/ttyd
```
2. Run tty
```bash
limactl shell ansible-rocky9 -- ttyd -p 7681 -W bash
```

3. Create service to persist terminal in web UI

---

## Architecture

```
limavm-tui/
├── cmd/
│   └── limavm-tui/       # main.go: CLI flags, store initialization, Bubble Tea program runner
├── internal/
│   ├── limactl/          # Typed wrapper for all limactl CLI commands via os/exec
│   │   ├── exec.go       # Command execution, context timeout, NDJSON stream decoder
│   │   ├── types.go      # Typed data structures (Instance, Disk, Network, Snapshot, Info)
│   │   ├── instance.go   # VM lifecycle commands (list, start, stop, delete, clone, rename)
│   │   ├── disk.go       # Disk commands (list, create, delete, resize, unlock)
│   │   ├── network.go    # Virtual network commands (list, create, delete)
│   │   ├── snapshot.go   # Snapshot commands (list, create, apply, delete)
│   │   ├── template.go   # Template discovery and description extraction from comments
│   │   └── misc.go       # Diagnostic info, prune, and YAML validation
│   ├── profile/          # Generic VM profile mechanism (external harness integration)
│   │   ├── profile.go    # Schema, path containment, placeholder expansion
│   │   ├── loader.go     # YAML loading and validation
│   │   └── discovery.go  # --profile sources and optional user config file
│   ├── ui/
│   │   ├── app.go        # Root model: screen routing, key dispatch, modal overlays, ExecProcess
│   │   ├── common/       # Messages and cross-screen events (zero circular dependencies)
│   │   ├── dashboard/    # Main VM table view with live polling and filter
│   │   ├── detail/       # 5-tab VM inspection view (Overview, Disks, Network, Snapshots, YAML)
│   │   ├── createvm/     # Creation form: template picker with preview & custom YAML path
│   │   ├── networks/     # Dedicated network management view
│   │   ├── profiles/     # Profiles screen: harness lifecycle and provisioning modes
│   │   └── components/   # HelpBar, Toast notifications, ConfirmDialog, PromptDialog, Spinner, Modals
│   ├── theme/            # Unified Lip Gloss dark palette (borders, badges, selection, toasts)
│   └── store/            # Atomic JSON persistence for favorites and recent preferences
├── Makefile              # make build / test / vet / install
└── Dockerfile            # CI cross-compilation container (build-only)
```

---

## License

Apache 2.0 or MIT.
