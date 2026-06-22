package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"gdctl/internal/bridge"
)

func runAsset(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("asset requires a subcommand: scan, missing, unused")
	}
	switch args[0] {
	case "scan":
		return runAssetScan(ctx, client, args[1:], stdout)
	case "missing":
		return runAssetMissing(ctx, client, args[1:], stdout)
	case "unused":
		fmt.Fprintln(stdout, "asset unused: requires scene reference parsing — not yet available.")
		fmt.Fprintln(stdout, "Hint: bridge support for reading .tscn resource references is needed.")
		return nil
	}
	return fmt.Errorf("unknown asset subcommand: %s", args[0])
}

// runAssetScan enumerates all project files and groups them by extension category.
func runAssetScan(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("asset scan")
	dir := fs.String("dir", "res://", "project directory to scan")
	jsonOut := fs.Bool("json", false, "print result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := client.FileList(ctx, requestID(), *dir, true)
	if err != nil {
		return err
	}

	buckets := categorizeFiles(result.Files)

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		total := 0
		for _, files := range buckets {
			total += len(files)
		}
		out := map[string]any{
			"dir":    *dir,
			"total":  total,
			"groups": map[string]int{},
		}
		groups := out["groups"].(map[string]int)
		for cat, files := range buckets {
			groups[cat] = len(files)
		}
		return enc.Encode(out)
	}

	fmt.Fprintf(stdout, "Asset scan: %s\n", *dir)
	cats := []string{"scripts", "scenes", "textures", "resources", "shaders", "audio", "other"}
	total := 0
	for _, cat := range cats {
		files := buckets[cat]
		if len(files) == 0 {
			continue
		}
		exts := extensionLabel(cat)
		fmt.Fprintf(stdout, "  %-26s %d files\n", exts+":", len(files))
		total += len(files)
	}
	fmt.Fprintf(stdout, "  %-26s %d files\n", "Total:", total)
	return nil
}

// runAssetMissing checks that all autoload paths exist in the project.
func runAssetMissing(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("asset missing")
	jsonOut := fs.Bool("json", false, "print result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	al, err := client.AutoloadList(ctx, requestID())
	if err != nil {
		return err
	}

	type missingEntry struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	var missing []missingEntry

	for _, a := range al.Autoloads {
		if a.Path == "" {
			continue
		}
		exists, err := client.FileExists(ctx, requestID(), a.Path)
		if err != nil {
			continue
		}
		if !exists.Exists {
			missing = append(missing, missingEntry{Name: a.Name, Path: a.Path})
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"checked": len(al.Autoloads),
			"missing": missing,
		})
	}

	fmt.Fprintf(stdout, "Asset missing (autoloads checked: %d)\n", len(al.Autoloads))
	if len(missing) == 0 {
		fmt.Fprintln(stdout, "  No missing autoload paths.")
	} else {
		for _, m := range missing {
			fmt.Fprintf(stdout, "  MISSING  %s → %s\n", m.Name, m.Path)
		}
		fmt.Fprintf(stdout, "\n%d missing path(s) found.\n", len(missing))
	}
	fmt.Fprintln(stdout, "\nNote: full scene reference scanning requires bridge support not yet available.")
	return nil
}

// categorizeFiles groups file paths by extension category.
func categorizeFiles(files []string) map[string][]string {
	buckets := map[string][]string{
		"scripts":   {},
		"scenes":    {},
		"textures":  {},
		"resources": {},
		"shaders":   {},
		"audio":     {},
		"other":     {},
	}
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		switch ext {
		case ".gd", ".gdscript":
			buckets["scripts"] = append(buckets["scripts"], f)
		case ".tscn", ".scn":
			buckets["scenes"] = append(buckets["scenes"], f)
		case ".png", ".jpg", ".jpeg", ".webp", ".exr", ".hdr", ".svg", ".bmp", ".tga":
			buckets["textures"] = append(buckets["textures"], f)
		case ".tres", ".res", ".material", ".mesh", ".theme", ".font", ".fontdata":
			buckets["resources"] = append(buckets["resources"], f)
		case ".gdshader", ".glsl", ".shader":
			buckets["shaders"] = append(buckets["shaders"], f)
		case ".wav", ".ogg", ".mp3", ".opus":
			buckets["audio"] = append(buckets["audio"], f)
		default:
			buckets["other"] = append(buckets["other"], f)
		}
	}
	// Sort each bucket for deterministic output
	for _, v := range buckets {
		sort.Strings(v)
	}
	return buckets
}

func extensionLabel(cat string) string {
	switch cat {
	case "scripts":
		return "Scripts (.gd)"
	case "scenes":
		return "Scenes (.tscn)"
	case "textures":
		return "Textures (.png .webp …)"
	case "resources":
		return "Resources (.tres .res)"
	case "shaders":
		return "Shaders (.gdshader)"
	case "audio":
		return "Audio (.wav .ogg)"
	case "other":
		return "Other"
	}
	return cat
}
