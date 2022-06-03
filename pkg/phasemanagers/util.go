package phasemanagers

import (
	"fmt"
	"strings"

	rel "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/storage"
)

func ReleaseHistoryUntilRevision(name string, ignoreSinceRevision int, storage *storage.Storage) ([]*rel.Release, error) {
	history, err := storage.History(name)
	if err != nil {
		return nil, fmt.Errorf("error getting release history: %w", err)
	}

	releaseutil.SortByRevision(history)

	resultLength := len(history)
	for i, release := range history {
		if release.Version == ignoreSinceRevision {
			resultLength = i
			break
		}
	}

	return history[:resultLength], nil
}

func joinErrors(errs []error) string {
	es := make([]string, 0, len(errs))
	for _, e := range errs {
		es = append(es, e.Error())
	}

	return strings.Join(es, "; ")
}
