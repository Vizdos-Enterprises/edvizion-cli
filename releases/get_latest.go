package releases

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/vizdos-enterprises/edvizion-cli/domain"
)

func GetLatestReleaseID(
	ctx context.Context,
	tool domain.Tool,
) (*int64, string, error) {
	owner, name := ParseRepository(tool.Repository)
	releases, _, err := client.Repositories.ListReleases(
		ctx,
		owner,
		name,
		nil,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list releases: %w", err)
	}

	var (
		latestVersion *semver.Version
		latestID      *int64
	)

	for _, release := range releases {
		tag := release.GetTagName()

		if !strings.HasPrefix(tag, tool.ReleasePrefix) {
			continue
		}

		versionString := strings.TrimPrefix(tag, tool.ReleasePrefix)

		version, err := semver.NewVersion(versionString)
		if err != nil {
			continue
		}

		if latestVersion == nil || version.GreaterThan(latestVersion) {
			latestVersion = version

			id := release.GetID()
			latestID = &id
		}
	}

	return latestID, latestVersion.String(), nil
}
