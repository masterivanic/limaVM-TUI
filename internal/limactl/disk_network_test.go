package limactl

import (
	"strings"
	"testing"
)

func TestDecodeDiskJSON(t *testing.T) {
	input := `{"name":"data-disk","size":10737418240,"format":"qcow2","dir":"/home/user/.lima/_disks/data-disk","instance":"ansible-rocky9","mountPath":"/mnt/data"}
{"name":"scratch","size":5368709120,"format":"raw","dir":"/home/user/.lima/_disks/scratch","locked":false}
`
	disks, err := DecodeNDJSON[Disk](strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(disks))
	}
	if disks[0].Name != "data-disk" || disks[0].Instance != "ansible-rocky9" {
		t.Errorf("disk 0 mismatch: %+v", disks[0])
	}
	if disks[1].Name != "scratch" || disks[1].Format != "raw" {
		t.Errorf("disk 1 mismatch: %+v", disks[1])
	}
}

func TestDecodeNetworkJSON(t *testing.T) {
	input := `{"mode":"bridged","interface":"en0"}
{"mode":"host","gateway":"192.168.106.1","dhcpEnd":"192.168.106.254","netmask":"255.255.255.0"}
{"name":"shared","mode":"shared","gateway":"192.168.105.1","netmask":"255.255.255.0"}
`
	nets, err := DecodeNDJSON[Network](strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nets) != 3 {
		t.Fatalf("expected 3 networks, got %d", len(nets))
	}
	if nets[0].Mode != "bridged" || nets[0].Interface != "en0" {
		t.Errorf("net 0 mismatch: %+v", nets[0])
	}
	if nets[2].Name != "shared" || nets[2].Gateway != "192.168.105.1" {
		t.Errorf("net 2 mismatch: %+v", nets[2])
	}
}
