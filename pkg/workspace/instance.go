package workspace

// InstanceStatus is derived from disk, never stored.
type InstanceStatus string

const (
	InstanceStatusNew          InstanceStatus = "new"
	InstanceStatusArchived     InstanceStatus = "archived"
	InstanceStatusPartial      InstanceStatus = "partial"
	InstanceStatusMaterialized InstanceStatus = "materialized"
)

const InstanceStatusNotMaterialized = InstanceStatusNew

type MemberState string

const (
	MemberStateUnknown MemberState = "—"
	MemberStateClean   MemberState = "clean"
	MemberStateDirty   MemberState = "dirty"
)

type InstanceMember = Member

// Instance is a workspace discovered on disk.
type Instance struct {
	Name    string // directory name under the workspaces root
	Dir     string // absolute path to the instance directory
	Members []InstanceMember
	Status  InstanceStatus
}
