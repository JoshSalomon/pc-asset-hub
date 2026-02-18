package validation

import (
	"context"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// CheckContainmentCycle checks if adding a containment edge from sourceEntityTypeID
// to targetEntityTypeID would create a cycle in the containment DAG.
func CheckContainmentCycle(ctx context.Context, assocRepo repository.AssociationRepository, sourceEntityTypeID, targetEntityTypeID string) error {
	if sourceEntityTypeID == targetEntityTypeID {
		return domainerrors.NewCycleDetected("entity type cannot contain itself")
	}

	edges, err := assocRepo.GetContainmentGraph(ctx)
	if err != nil {
		return err
	}

	// Build adjacency list
	adj := make(map[string][]string)
	for _, e := range edges {
		adj[e.SourceEntityTypeID] = append(adj[e.SourceEntityTypeID], e.TargetEntityTypeID)
	}

	// Add the proposed edge
	adj[sourceEntityTypeID] = append(adj[sourceEntityTypeID], targetEntityTypeID)

	// DFS from targetEntityTypeID to see if we can reach sourceEntityTypeID
	// (which would mean a cycle: source -> target -> ... -> source)
	visited := make(map[string]bool)
	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		if node == sourceEntityTypeID {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		for _, next := range adj[node] {
			if hasCycle(next) {
				return true
			}
		}
		return false
	}

	if hasCycle(targetEntityTypeID) {
		return domainerrors.NewCycleDetected("adding this containment would create a cycle")
	}

	return nil
}
