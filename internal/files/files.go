package files

import (
	"context"

	"github.com/anfragment/zen/internal/cfg"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Files struct {
	ctx context.Context
	cfg *cfg.Config
}

func NewFiles() *Files {
	return &Files{}
}

func (f *Files) Init(ctx context.Context, cfg *cfg.Config) {
	f.ctx = ctx
	f.cfg = cfg
}

// ExportFilterList exports the custom filter lists to a file.
func (f *Files) ExportFilterList() error {
	filePath, err := runtime.SaveFileDialog(f.ctx, runtime.SaveDialogOptions{
		Title:           "Export Custom Filter Lists",
		DefaultFilename: "filter-lists.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})

	if err != nil {
		return err
	}

	return f.cfg.ExportFilterListToFile(filePath)
}

// ImportFilterList imports the custom filter lists from a file.
func (f *Files) ImportFilterList() error {
	filePath, err := runtime.OpenFileDialog(f.ctx, runtime.OpenDialogOptions{
		Title: "Import Custom Filter Lists",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})

	if err != nil {
		return err
	}

	return f.cfg.ImportFilterList(filePath)
}
