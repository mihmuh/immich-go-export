package export

import (
	"context"
	"time"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/spf13/cobra"
)

type ExportCmd struct {
	app         *app.Application
	Dest        string
	DryRun      bool
	DateInPath  bool
	IgnoreFiles []string
}

func NewExportCommand(ctx context.Context, app *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export photos from various sources to local disk",
		Args:  cobra.NoArgs,
	}

	ec := &ExportCmd{app: app}
	cmd.PersistentFlags().StringVar(&ec.Dest, "dest", ".", "Destination directory for exported files")

	// Add subcommands
	cmd.AddCommand(gp.NewFromGooglePhotosCommand(ctx, cmd, app, ec))

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Initialize the FileProcessor (tracker + logger)
		if app.FileProcessor() == nil {
			recorder := fileevent.NewRecorder(app.Log().Logger)
			tracker := assettracker.NewWithLogger(app.Log().Logger, app.DryRun) // Enable debug mode in dry-run
			processor := fileprocessor.New(tracker, recorder)
			app.SetFileProcessor(processor)
		}
		app.SetTZ(time.Local)
		return nil
	}

	return cmd
}
