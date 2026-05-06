package addon

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Project struct {
	Path         string
	ProjectGodot string
}

func ResolveProject(path string) (Project, error) {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Project{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Project{}, fmt.Errorf("project path does not exist: %s", abs)
	}
	if !info.IsDir() {
		return Project{}, fmt.Errorf("project path is not a directory: %s", abs)
	}
	projectGodot := filepath.Join(abs, "project.godot")
	if _, err := os.Stat(projectGodot); err != nil {
		if os.IsNotExist(err) {
			return Project{}, fmt.Errorf("not a Godot project, missing project.godot: %s", abs)
		}
		return Project{}, err
	}
	return Project{Path: abs, ProjectGodot: projectGodot}, nil
}

func (p Project) AddonPath() string {
	return filepath.Join(p.Path, AddonDir)
}

func (p Project) PluginPath() string {
	return filepath.Join(p.Path, AddonDir, "plugin.cfg")
}

func IsEnabled(projectFile string) (bool, error) {
	data, err := os.ReadFile(projectFile)
	if err != nil {
		return false, err
	}
	return containsEnabledPlugin(string(data), PluginResPath), nil
}

func SetEnabled(projectFile string, enabled bool) (bool, error) {
	data, err := os.ReadFile(projectFile)
	if err != nil {
		return false, err
	}
	updated, changed := setEnabledPlugin(string(data), PluginResPath, enabled)
	if !changed {
		return false, nil
	}
	return true, os.WriteFile(projectFile, []byte(updated), 0o644)
}

func containsEnabledPlugin(content, plugin string) bool {
	plugins, ok := readEnabledPlugins(content)
	if !ok {
		return false
	}
	for _, item := range plugins {
		if item == plugin {
			return true
		}
	}
	return false
}

func setEnabledPlugin(content, plugin string, enabled bool) (string, bool) {
	lines := strings.SplitAfter(content, "\n")
	plugins, lineIndex, found := readEnabledPluginLine(lines)
	if !found {
		if !enabled {
			return content, false
		}
		insert := "[editor_plugins]\n\nenabled=PackedStringArray(\"" + plugin + "\")\n"
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		return content + insert, true
	}

	hadPlugin := false
	next := make([]string, 0, len(plugins)+1)
	for _, item := range plugins {
		if item == plugin {
			hadPlugin = true
			if enabled {
				next = append(next, item)
			}
			continue
		}
		next = append(next, item)
	}
	if enabled && !hadPlugin {
		next = append(next, plugin)
	}
	if enabled == hadPlugin {
		return content, false
	}
	newLine := "enabled=" + formatPackedStringArray(next)
	if strings.HasSuffix(lines[lineIndex], "\n") {
		newLine += "\n"
	}
	lines[lineIndex] = newLine
	return strings.Join(lines, ""), true
}

func readEnabledPlugins(content string) ([]string, bool) {
	lines := strings.SplitAfter(content, "\n")
	plugins, _, ok := readEnabledPluginLine(lines)
	return plugins, ok
}

func readEnabledPluginLine(lines []string) ([]string, int, bool) {
	inSection := false
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inSection = line == "[editor_plugins]"
			continue
		}
		if !inSection || !strings.HasPrefix(line, "enabled=") {
			continue
		}
		return parsePackedStringArray(strings.TrimSpace(strings.TrimPrefix(line, "enabled="))), i, true
	}
	return nil, -1, false
}

func parsePackedStringArray(value string) []string {
	re := regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
	matches := re.FindAllStringSubmatch(value, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.ReplaceAll(match[1], `\"`, `"`))
	}
	return out
}

func formatPackedStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, `"`+strings.ReplaceAll(value, `"`, `\"`)+`"`)
	}
	return "PackedStringArray(" + strings.Join(quoted, ", ") + ")"
}
