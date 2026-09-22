package limactl

import (
	"fmt"
	"time"
)

// Instance represents a Lima VM instance parsed from `limactl list --json`.
type Instance struct {
	Name            string                 `json:"name"`
	Hostname        string                 `json:"hostname"`
	Status          string                 `json:"status"`
	Dir             string                 `json:"dir"`
	VMType          string                 `json:"vmType"`
	Arch            string                 `json:"arch"`
	CPUType         string                 `json:"cpuType"`
	CPUs            int                    `json:"cpus"`
	Memory          int64                  `json:"memory"` // bytes
	Disk            int64                  `json:"disk"`   // bytes
	SSHConfigFile   string                 `json:"sshConfigFile"`
	SSHAddress      string                 `json:"sshAddress"`
	SSHLocalPort    int                    `json:"sshLocalPort"`
	Protected       bool                   `json:"protected"`
	LimaVersion     string                 `json:"limaVersion"`
	HostOS          string                 `json:"HostOS"`
	HostArch        string                 `json:"HostArch"`
	LimaHome        string                 `json:"LimaHome"`
	IdentityFile    string                 `json:"IdentityFile"`
	Errors          []string               `json:"errors,omitempty"`
	Networks        []InstanceNetwork      `json:"networks,omitempty"`
	AdditionalDisks []InstanceDiskRef      `json:"additionalDisks,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
}

// InstanceNetwork represents an attached network configuration in an instance.
type InstanceNetwork struct {
	Lima      string `json:"lima,omitempty"`
	MAC       string `json:"macAddress,omitempty"`
	Interface string `json:"interface,omitempty"`
}

// InstanceDiskRef represents a reference to an attached disk in an instance.
type InstanceDiskRef struct {
	Name   string `json:"name"`
	Format string `json:"format,omitempty"`
}

// Disk represents a disk managed by limactl (from `limactl disk list --json`).
type Disk struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"` // bytes
	Format    string `json:"format"`
	Dir       string `json:"dir"`
	Instance  string `json:"instance,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
	Locked    bool   `json:"locked,omitempty"`
}

// Network represents a Lima virtual network (from `limactl network list --json`).
type Network struct {
	Name      string `json:"name,omitempty"`
	Mode      string `json:"mode"` // user-v2, host, shared, bridged
	Gateway   string `json:"gateway,omitempty"`
	Netmask   string `json:"netmask,omitempty"`
	DHCPStart string `json:"dhcpStart,omitempty"`
	DHCPEnd   string `json:"dhcpEnd,omitempty"`
	Interface string `json:"interface,omitempty"`
}

// Snapshot represents a point-in-time snapshot of an instance.
type Snapshot struct {
	Tag     string    `json:"tag"`
	Created time.Time `json:"created,omitempty"`
	Size    string    `json:"size,omitempty"`
	Parent  string    `json:"parent,omitempty"`
}

// InfoDiagnostic represents diagnostic output from `limactl info`.
type InfoDiagnostic struct {
	Version         string                 `json:"version"`
	DefaultTemplate interface{}            `json:"defaultTemplate,omitempty"`
	LimaHome        string                 `json:"limaHome"`
	VMTypes         []string               `json:"vmTypes"`
	GuestAgents     map[string]interface{} `json:"guestAgents,omitempty"`
	Templates       []TemplateLocation     `json:"templates"`
}

// TemplateLocation describes a template returned by `limactl info`.
type TemplateLocation struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

// Template represents an enriched template with metadata extracted from comments.
type Template struct {
	Name        string   `json:"name"`
	Location    string   `json:"location"`
	Description string   `json:"description"`
	ArchList    []string `json:"arch_list,omitempty"`
	IsDefault   bool     `json:"is_default"`
}

// HumanMemory formats a byte count into a readable string (e.g. 2.0 GiB).
func HumanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	if exp < len(suffixes) {
		return fmt.Sprintf("%.1f %s", float64(b)/float64(div), suffixes[exp])
	}
	return fmt.Sprintf("%d B", b)
}
