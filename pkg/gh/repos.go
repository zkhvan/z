package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type Repo struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

func (r *Repo) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

type RepoListOptions struct {
	Owner string
}

func (c *Client) ListRepos(ctx context.Context, opts *RepoListOptions) ([]*Repo, error) {
	if opts == nil {
		opts = &RepoListOptions{}
	}

	if opts.Owner == "" {
		return nil, errors.New("owner is required")
	}

	cmd := c.executor.CommandContext(
		ctx,
		"gh", "repo", "list",
		opts.Owner, "--json", "owner,name",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}

	output = bytes.TrimSpace(output)

	type nameOwner struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	}

	var nameOwners []*nameOwner
	if err := json.Unmarshal(output, &nameOwners); err != nil {
		return nil, fmt.Errorf("error unmarshalling: %w", err)
	}

	var repos []*Repo
	for _, r := range nameOwners {
		repos = append(repos, &Repo{
			Owner: r.Owner.Login,
			Name:  r.Name,
		})
	}

	return repos, nil
}
