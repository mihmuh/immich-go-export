package export

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"testing/fstest"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdapter struct {
	groups []*assets.Group
}

func (m *mockAdapter) Browse(ctx context.Context) chan *assets.Group {
	c := make(chan *assets.Group)
	go func() {
		defer close(c)
		for _, g := range m.groups {
			select {
			case <-ctx.Done():
				return
			case c <- g:
			}
		}
	}()
	return c
}

func TestExportCmd_Run(t *testing.T) {
	// Setup
	tmpDest := t.TempDir()

	// Create a minimal app with logger
	application := &app.Application{}
	application.SetLog(&app.Log{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Level:  "INFO",
	})

	ec := &ExportCmd{
		app:  application,
		Dest: tmpDest,
	}

	// Create a mock asset
	capturedDate := time.Date(2023, 10, 25, 14, 0, 0, 0, time.UTC)
	testContent := []byte("fake image content")

	// Create a memory FS to simulate source file
	memFS := fstest.MapFS{
		"test.jpg": &fstest.MapFile{
			Data:    testContent,
			ModTime: time.Time{},
		},
	}

	// fshelper.FSName creates a FSAndName (assuming based on usage in other files I saw)
	// I saw usage: fshelper.FSName(w, name) in googlephotos.go

	asset := &assets.Asset{
		File:             fshelper.FSName(memFS, "test.jpg"),
		OriginalFileName: "photo.jpg",
		CaptureDate:      capturedDate,
		FileSize:         len(testContent),
	}

	group := assets.NewGroup(assets.GroupByNone, asset)

	adapter := &mockAdapter{
		groups: []*assets.Group{group},
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := ec.Run(cmd, adapter)
	require.NoError(t, err)

	// Verify file exists
	expectedPath := filepath.Join(tmpDest, "2023", "10", "photo.jpg")
	info, err := os.Stat(expectedPath)
	require.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, content)

	// Verify mtime
	assert.True(t, info.ModTime().Equal(capturedDate), "mtime should match captured date")
}
