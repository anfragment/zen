package files

import (
	"context"

	"github.com/anfragment/zen/internal/cfg"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type FileExport struct {
	ctx context.Context
	cfg *cfg.Config
}

func NewFileExport() *FileExport {
	return &FileExport{}
}
func (e *FileExport) Init(ctx context.Context, cfg *cfg.Config) {
	e.ctx = ctx
	e.cfg = cfg
}

func (e *FileExport) ExportFilterList() error {
	filePath, err := runtime.SaveFileDialog(e.ctx, runtime.SaveDialogOptions{
		Title:           "Export Custom Filter Lists",
		DefaultFilename: "filter-lists.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})

	if err != nil {
		return err
	}

	return e.cfg.ExportFilterListToFile(filePath)
}
