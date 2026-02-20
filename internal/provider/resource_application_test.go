package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeApplicationMountPlan_DefaultsMountTypeToVolume(t *testing.T) {
	mount := normalizeApplicationMountPlan(ApplicationMountResourceModel{
		MountType:  types.StringNull(),
		MountPath:  types.StringValue("/data"),
		VolumeName: types.StringValue("rssmate-sqlite-data"),
	})

	if mount.MountType != "volume" {
		t.Fatalf("unexpected mount type: got %q want %q", mount.MountType, "volume")
	}
	if mount.MountPath != "/data" {
		t.Fatalf("unexpected mount path: got %q want %q", mount.MountPath, "/data")
	}
	if mount.VolumeName != "rssmate-sqlite-data" {
		t.Fatalf("unexpected volume name: got %q want %q", mount.VolumeName, "rssmate-sqlite-data")
	}
}

func TestNormalizeApplicationMountPlan_TrimsValues(t *testing.T) {
	mount := normalizeApplicationMountPlan(ApplicationMountResourceModel{
		MountType:  types.StringValue(" volume "),
		MountPath:  types.StringValue(" /data "),
		VolumeName: types.StringValue(" rssmate-sqlite-data "),
	})

	if mount.MountType != "volume" {
		t.Fatalf("unexpected mount type: got %q want %q", mount.MountType, "volume")
	}
	if mount.MountPath != "/data" {
		t.Fatalf("unexpected mount path: got %q want %q", mount.MountPath, "/data")
	}
	if mount.VolumeName != "rssmate-sqlite-data" {
		t.Fatalf("unexpected volume name: got %q want %q", mount.VolumeName, "rssmate-sqlite-data")
	}
}
