package releases

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/google/go-github/v90/github"
	"github.com/vizdos-enterprises/edvizion-cli/domain"
)

var client *github.Client

func init() {
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		panic(err)
	}

	client, err = github.NewClient(
		github.WithHTTPClient(httpClient),
	)
	if err != nil {
		panic(err)
	}
}

func Download(ctx context.Context, tool domain.Tool, releaseID int64, outPath string) error {
	owner, repo := ParseRepository(tool.Repository)

	assets, _, err := client.Repositories.ListReleaseAssets(
		ctx,
		owner,
		repo,
		releaseID,
		nil,
	)
	if err != nil {
		return err
	}

	expectName := fmt.Sprintf(
		"%s-%s-%s",
		tool.BinaryName,
		runtime.GOOS,
		runtime.GOARCH,
	)

	if runtime.GOOS == "windows" {
		expectName += ".exe"
	}

	var asset *github.ReleaseAsset

	for _, a := range assets {
		if a.GetName() == expectName {
			asset = a
			break
		}
	}

	if asset == nil {
		return fmt.Errorf("no asset found for %s", expectName)
	}

	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(
		ctx,
		owner,
		repo,
		asset.GetID(),
		http.DefaultClient,
	)
	if err != nil {
		return err
	}

	if rc == nil {
		return fmt.Errorf("unexpected redirect to %s", redirectURL)
	}
	defer rc.Close()

	out, err := os.OpenFile(
		outPath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0755,
	)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return err
	}

	// Ensure executable bit, primarily for macOS/Linux.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(outPath, 0755); err != nil {
			return err
		}
	}

	return nil
}
