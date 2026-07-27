package wssync

// Mode selects how divergence is resolved. Only mutagen's unidirectional modes
// apply: z never writes back to a definition.
type Mode uint8

const (
	// ModeSafe lets the definition win unless doing so would destroy instance
	// content that was never synchronized.
	ModeSafe Mode = iota
	// ModeReplica makes the instance an exact mirror of the definition. Ignored
	// content is still untouchable; that is what keeps worktrees safe.
	ModeReplica
)

// Change transforms the instance (or the ancestor) at Path. A nil New is a
// deletion.
type Change struct {
	Path string
	Old  *Entry
	New  *Entry
}

// Deletion reports whether the change removes content.
func (c Change) Deletion() bool { return c.New == nil }

// Conflict is a path where the definition and the instance both changed since
// the ancestor. Resolution is manual: delete the side that should lose, since
// deletions are always safe to overwrite.
type Conflict struct {
	Path       string
	Definition *Entry
	Instance   *Entry
}

// Result is the plan. Changes are applied to the instance; AncestorChanges are
// applied to the sync state regardless of whether anything is written, which is
// how convergence and untracking are recorded.
type Result struct {
	Changes         []Change
	AncestorChanges []Change
	Conflicts       []Conflict
}

// Empty reports whether the plan would neither write nor record anything.
func (r Result) Empty() bool {
	return len(r.Changes) == 0 && len(r.AncestorChanges) == 0
}

// Reconcile computes the plan without touching the filesystem. It is pure over
// the three trees, which is the whole reason this package exists.
func Reconcile(ancestor, definition, instance *Entry, mode Mode) Result {
	r := &reconciler{mode: mode}
	r.reconcile("", ancestor, definition, instance)
	return Result{Changes: r.changes, AncestorChanges: r.ancestorChanges, Conflicts: r.conflicts}
}

type reconciler struct {
	mode            Mode
	changes         []Change
	ancestorChanges []Change
	conflicts       []Conflict
}

func (r *reconciler) reconcile(path string, ancestor, definition, instance *Entry) {
	// The scan already reported why this path cannot be handled; a conflict
	// here would just say it a second time.
	if definition.kindIs(KindProblematic) || instance.kindIs(KindProblematic) {
		return
	}

	// Nothing to track and nothing to disagree about.
	if definition.nilOrUntracked() && instance.nilOrUntracked() {
		if ancestor != nil {
			r.ancestorChanges = append(r.ancestorChanges, Change{Path: path, Old: ancestor})
		}
		return
	}

	if definition.Equal(instance, false) {
		ancestorContents := ancestor.contents()

		// The two sides agree but the ancestor disagrees: they converged
		// independently. Recording that is what makes an interrupted sync
		// recover on the next run instead of reporting phantom conflicts, and
		// what stops a user's edit-to-match from being called a conflict.
		if !ancestor.Equal(definition, false) {
			r.ancestorChanges = append(r.ancestorChanges, Change{Path: path, Old: ancestor, New: definition.slim()})
			ancestorContents = nil
		}

		for _, name := range nameUnion(ancestorContents, definition.contents(), instance.contents()) {
			r.reconcile(join(path, name), ancestorContents[name], definition.contents()[name], instance.contents()[name])
		}
		return
	}

	if r.mode == ModeReplica {
		r.disagreeReplica(path, definition, instance)
		return
	}
	r.disagreeSafe(path, ancestor, definition, instance)
}

func (r *reconciler) disagreeSafe(path string, ancestor, definition, instance *Entry) {
	synchronizable := instance.synchronizable()

	// If the instance has only deleted content since z wrote it, the definition
	// can be applied: deletions are never worth preserving, and re-propagating
	// is what makes `rm <file> && z workspace sync` mean "give me the
	// definition's version".
	if len(nonDeletions(diff(path, ancestor, synchronizable))) == 0 {
		if len(diff(path, synchronizable, instance)) > 0 {
			r.conflicts = append(r.conflicts, Conflict{Path: path, Definition: definition, Instance: instance})
			return
		}
		r.changes = append(r.changes, Change{Path: path, Old: instance, New: definition.synchronizable()})
		return
	}

	// The instance holds edits, and the definition no longer ships this path.
	// Nothing can be propagated and nothing should be reported forever, so z
	// releases the file: it becomes the user's.
	if definition.nilOrUntracked() && (!ancestor.kindIs(KindDirectory) || !instance.kindIs(KindDirectory)) {
		if ancestor != nil {
			r.ancestorChanges = append(r.ancestorChanges, Change{Path: path, Old: ancestor})
		}
		return
	}

	r.conflicts = append(r.conflicts, Conflict{Path: path, Definition: definition, Instance: instance})
}

func (r *reconciler) disagreeReplica(path string, definition, instance *Entry) {
	// Ignored or problematic content cannot be removed, so it blocks the mirror
	// rather than being silently destroyed by --force.
	if len(diff(path, instance.synchronizable(), instance)) > 0 {
		r.conflicts = append(r.conflicts, Conflict{Path: path, Definition: definition, Instance: instance})
		return
	}
	r.changes = append(r.changes, Change{Path: path, Old: instance, New: definition.synchronizable()})
}

// diff describes what it would take to turn base into target.
func diff(path string, base, target *Entry) []Change {
	if base.Equal(target, false) {
		if !base.kindIs(KindDirectory) {
			return nil
		}
		var changes []Change
		for _, name := range nameUnion(base.contents(), target.contents()) {
			changes = append(changes, diff(join(path, name), base.contents()[name], target.contents()[name])...)
		}
		return changes
	}
	return []Change{{Path: path, Old: base, New: target}}
}

func nonDeletions(changes []Change) []Change {
	var filtered []Change
	for _, c := range changes {
		if !c.Deletion() {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// Apply folds changes into a tree, returning the new root. Used to advance the
// ancestor; callers pass only the changes that actually landed.
func Apply(root *Entry, changes []Change) *Entry {
	result := root.copy()
	for _, c := range changes {
		result = applyChange(result, splitPath(c.Path), c.New)
	}
	return result
}

func applyChange(root *Entry, segments []string, value *Entry) *Entry {
	if len(segments) == 0 {
		if value == nil {
			return nil
		}
		merged := value.copy()
		// Convergence records a parent directory slim and then descends, so an
		// empty replacement must not orphan children already recorded beneath it.
		// A replacement that carries contents is authoritative.
		if merged.Kind == KindDirectory && len(merged.Contents) == 0 && root.kindIs(KindDirectory) {
			merged.Contents = root.Contents
		}
		return merged
	}

	parent := root
	if !parent.kindIs(KindDirectory) {
		if value == nil {
			return root
		}
		parent = &Entry{Kind: KindDirectory}
	} else {
		parent = &Entry{Kind: KindDirectory, Contents: make(map[string]*Entry, len(root.Contents))}
		for name, child := range root.Contents {
			parent.Contents[name] = child
		}
	}

	name := segments[0]
	child := applyChange(parent.Contents[name], segments[1:], value)
	if child == nil {
		delete(parent.Contents, name)
	} else {
		if parent.Contents == nil {
			parent.Contents = make(map[string]*Entry)
		}
		parent.Contents[name] = child
	}
	return parent
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	var segments []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			segments = append(segments, path[start:i])
			start = i + 1
		}
	}
	return append(segments, path[start:])
}
