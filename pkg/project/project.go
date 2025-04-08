package project

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

type SourceType int

const (
	SourceTypeUnknown SourceType = iota
	SourceTypeLocal
	SourceTypeRemote
	SourceTypeSynced
)

func (s SourceType) String() string {
	switch s {
	case SourceTypeLocal:
		return "[L]"
	case SourceTypeRemote:
		return "[R]"
	case SourceTypeSynced:
		return "[S]"
	default:
		return "[?]"
	}
}

type Project struct {
	// LocalID is the identifier of the project on the local machine.
	//
	// For now, it's the relative path to the project from the root directory.
	LocalID string `json:"local_id"`

	// RemoteID is the identifier of the project on the remote service.
	//
	// For now, only GitHub is supported and this is usually the owner/repo.
	RemoteID string `json:"remote_id"`

	// AbsolutePath is the absolute path to the project.
	AbsolutePath string `json:"absolute_path"`

	// Source indicates how the project was discovered.
	Source SourceType `json:"source_type"`
}

// URL returns the URL of the project.
//
// This is a quick way to determine the URL based on the fact that all there's
// an assumption that all projects are GitHub repositories.
//
// TODO: Detect the proper URL in a generic way, based on the project type.
func (p Project) URL() string {
	owner, repo := p.OwnerRepo()
	return fmt.Sprintf("https://github.com/%s/%s", owner, repo)
}

func (p Project) OwnerRepo() (string, string) {
	owner := path.Dir(p.RemoteID)
	repo := path.Base(p.RemoteID)

	return owner, repo
}

func (p Project) Compare(other Project) int {
	return strings.Compare(p.AbsolutePath, other.AbsolutePath)
}

func newProject(localID, remoteID, abs string) Project {
	return Project{
		LocalID:      localID,
		RemoteID:     remoteID,
		AbsolutePath: abs,
	}
}

func (s *Service) Get(_ context.Context, id string) (Project, error) {
	var project Project

	parts := strings.Split(id, "/")

	switch n := len(parts); {
	case n < 2:
		return project, fmt.Errorf("invalid ID")
	case n == 2:
		project.RemoteID = id
		project.LocalID = s.toLocalID(id)
	case 2 < n:
		project.LocalID = id
		project.RemoteID = s.toRemoteID(id)
	}

	project.AbsolutePath = filepath.Join(s.cfg.Root, project.LocalID)

	return project, nil
}

func (s *Service) toRemoteID(localID string) string {
	// Convert a local ID to a remote ID.
	// A local ID is represented as the relative path to the project from the
	// root directory. The owner and repo can be extracted from the local ID
	// by analyzing the last two segments of the ID.

	owner := path.Base(path.Dir(localID))
	repo := path.Base(localID)

	return fmt.Sprintf("%s/%s", owner, repo)
}

// TODO: iterating over remote patterns isn't the most efficient, might want
// to make it lookup-based instead.
func (s *Service) toLocalID(remoteID string) string {
	parts := strings.Split(remoteID, "/")
	if len(parts) < 2 {
		return ""
	}

	owner := parts[0]
	repo := parts[1]

	localID := remoteID
	for _, pattern := range s.cfg.remotePatterns {
		if pattern.Owner != owner {
			continue
		}

		if pattern.Repo != "*" && pattern.Repo != repo {
			continue
		}

		// The alternate path might be empty, but filepath.Join will handle it
		// gracefully.
		localID = filepath.Join(pattern.AlternatePath, localID)
	}

	return localID
}
