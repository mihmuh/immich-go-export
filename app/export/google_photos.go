package export

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/spf13/cobra"
)

func (ec *ExportCmd) Run(cmd *cobra.Command, adapter adapters.Reader) error {
	ctx := cmd.Context()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(ec.Dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	app := ec.app
	groupChan := adapter.Browse(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case group, ok := <-groupChan:
			if !ok {
				return nil
			}
			if err := ec.handleGroup(ctx, group); err != nil {
				app.Log().Error("Error processing group", "err", err)
			}
		}
	}
}

func (ec *ExportCmd) handleGroup(ctx context.Context, group *assets.Group) error {
	for _, asset := range group.Assets {
		if err := ec.exportAsset(ctx, asset); err != nil {
			ec.app.Log().Error("Failed to export asset", "asset", asset.OriginalFileName, "err", err)
		}
	}
	return nil
}

func afterLastSlash(s string) string {
	i := strings.LastIndex(s, "/")
	if i == -1 {
		return s // no slash found
	}
	return s[i+1:]
}

func (ec *ExportCmd) exportAsset(ctx context.Context, asset *assets.Asset) error {

	if asset.Trashed {
		return nil // Skip trashed if they pass through (adapter might have filtered them already)
	}

	// Determine destination path
	captureDate := asset.CaptureDate
	if captureDate.IsZero() {
		captureDate = asset.FileDate
	}

	year := captureDate.Format("2006")
	month := captureDate.Format("01")

	destDir := filepath.Join(ec.Dest, year, month)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	destPath := correctName(asset, destDir)

	// Check idempotency
	if _, err := os.Stat(destPath); err == nil {
		ec.app.Log().Info("Skipping existing file", "path", destPath, "original", asset.OriginalFileName, "file", afterLastSlash(asset.File.Name()))
		return nil
	}

	// Copy content
	r, err := asset.File.Open()
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer r.Close()

	w, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	// Restore mtime
	if !captureDate.IsZero() {
		if err := os.Chtimes(destPath, time.Now(), captureDate); err != nil {
			ec.app.Log().Warn("Failed to set file dates", "path", destPath, "err", err)
		}
	}

	ec.app.Log().Info("Exported", "path", destPath)
	return nil
}

func correctName(asset *assets.Asset, destDir string) string {
	fname := afterLastSlash(asset.File.Name())
	if fname != asset.OriginalFileName && strings.Contains(fname, "-edited") {
		return filepath.Join(destDir, fname)
	} else {
		return filepath.Join(destDir, asset.OriginalFileName)
	}
}
