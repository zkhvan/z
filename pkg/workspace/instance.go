package workspace

// InstanceStatus is derived from disk, never stored.
type InstanceStatus string

// InstanceStatusNotMaterialized means no worktrees are checked out.
const InstanceStatusNotMaterialized InstanceStatus = "—"

// Instance is a workspace discovered on disk.
type Instance struct {
	Name    string // directory name under the workspaces root
	Dir     string // absolute path to the instance directory
	Members []Member
	Status  InstanceStatus
}
