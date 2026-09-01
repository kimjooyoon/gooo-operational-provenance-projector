package provenance

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func inventory(root, output string) (InventoryReport, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil { return InventoryReport{}, err }
	outputAbs, err := filepath.Abs(output)
	if err != nil { return InventoryReport{}, err }
	report := InventoryReport{RootReadmeExcluded: true, GitExcluded: true, OutputExcluded: true}
	err = filepath.WalkDir(rootAbs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil { return walkErr }
		if path == rootAbs { return nil }
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil { return err }
		if entry.IsDir() {
			if rel == ".git" || rel == "vendor" || rel == ".cache" || isWithin(path, outputAbs) { return fs.SkipDir }
			report.DescendantDirs++
			return nil
		}
		if rel == "README.md" || rel == ".git" || isWithin(path, outputAbs) || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) || strings.HasPrefix(rel, "vendor"+string(filepath.Separator)) || strings.HasPrefix(rel, ".cache"+string(filepath.Separator)) {
			return nil
		}
		report.RegularFiles++
		switch filepath.Ext(path) {
		case ".go": report.GoFiles++
		case ".gooo": report.GoooFiles++
		}
		return nil
	})
	return report, err
}

func isWithin(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

