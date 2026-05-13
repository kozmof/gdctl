package bridge

import (
	"fmt"
	"strings"
)

type PingResponse struct {
	OK              bool     `json:"ok"`
	Service         string   `json:"service"`
	Engine          string   `json:"engine"`
	EngineVersion   string   `json:"engine_version"`
	PluginVersion   string   `json:"plugin_version"`
	ProjectName     string   `json:"project_name"`
	ProjectPath     string   `json:"project_path"`
	SceneOpen       bool     `json:"scene_open"`
	AuthEnabled     bool     `json:"auth_enabled"`
	Host            string   `json:"host"`
	Port            int      `json:"port"`
	ProtocolVersion string   `json:"protocol_version"`
	Capabilities    []string `json:"capabilities"`
}

type BridgeError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

func (e *BridgeError) Error() string {
	if e == nil {
		return ""
	}
	var base string
	if e.Code == "" {
		base = e.Message
	} else {
		base = e.Code + ": " + e.Message
	}
	if len(e.Detail) == 0 {
		return base
	}
	var suffix []string
	path, _ := e.Detail["path"].(string)
	line := detailInt(e.Detail["line"])
	if path != "" && line > 0 {
		suffix = append(suffix, fmt.Sprintf("%s:%d", path, line))
	} else if path != "" {
		suffix = append(suffix, path)
	} else if line > 0 {
		suffix = append(suffix, fmt.Sprintf("line %d", line))
	}
	if diagnostic, _ := e.Detail["diagnostic"].(string); diagnostic != "" {
		suffix = append(suffix, diagnostic)
	} else if errText, _ := e.Detail["error"].(string); errText != "" {
		suffix = append(suffix, errText)
	}
	if hint, _ := e.Detail["hint"].(string); hint != "" {
		suffix = append(suffix, "hint: "+hint)
	}
	if len(suffix) > 0 {
		base += " (" + strings.Join(suffix, ": ") + ")"
	}
	if debugger := formatDebuggerContext(e.Detail["debugger"]); debugger != "" {
		base += " (" + debugger + ")"
	}
	if source := formatSourceContext(e.Detail["source"]); source != "" {
		base += "\n" + source
	}
	return base
}

func detailInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case jsonNumber:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

type jsonNumber interface {
	Int64() (int64, error)
}

func formatSourceContext(value any) string {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		line := detailInt(entry["line"])
		text, _ := entry["text"].(string)
		marker := " "
		if isErrorLine, _ := entry["error"].(bool); isErrorLine {
			marker = ">"
		}
		lines = append(lines, fmt.Sprintf("%s %4d | %s", marker, line, text))
	}
	return strings.Join(lines, "\n")
}

func formatDebuggerContext(value any) string {
	detail, ok := value.(map[string]any)
	if !ok || len(detail) == 0 {
		return ""
	}
	if paused, _ := detail["paused"].(bool); !paused {
		return ""
	}
	file, _ := detail["file"].(string)
	message, _ := detail["message"].(string)
	line := detailInt(detail["line"])
	var parts []string
	if file != "" && line > 0 {
		parts = append(parts, fmt.Sprintf("debugger paused at %s:%d", file, line))
	} else if file != "" {
		parts = append(parts, "debugger paused at "+file)
	} else {
		parts = append(parts, "debugger paused")
	}
	if message != "" {
		parts = append(parts, message)
	}
	return strings.Join(parts, ": ")
}

type BridgeResponse[T any] struct {
	RequestID string       `json:"request_id,omitempty"`
	OK        bool         `json:"ok"`
	Result    T            `json:"result,omitempty"`
	Error     *BridgeError `json:"error,omitempty"`
}

type NodeInfo struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Path     string     `json:"path"`
	Children []NodeInfo `json:"children"`
}

type SceneTreeResponse struct {
	OK   bool     `json:"ok"`
	Root NodeInfo `json:"root"`
}

type RequestEnvelope struct {
	RequestID string         `json:"request_id"`
	Op        string         `json:"op"`
	Params    map[string]any `json:"params"`
}

type AddonUpdateFile struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
}

type AddonUpdateResult struct {
	Updated        bool   `json:"updated"`
	FilesWritten   int    `json:"files_written"`
	FilesRemoved   int    `json:"files_removed,omitempty"`
	Backup         string `json:"backup,omitempty"`
	ReloadRequired bool   `json:"reload_required"`
}

type SceneSaveResult struct {
	Path   string `json:"path,omitempty"`
	Root   string `json:"root,omitempty"`
	Saved  bool   `json:"saved,omitempty"`
	Queued bool   `json:"queued,omitempty"`
	JobID  string `json:"job_id,omitempty"`
}

type SceneCreateResult struct {
	Path     string `json:"path"`
	RootType string `json:"root_type"`
	RootName string `json:"root_name"`
	RootPath string `json:"root_path"`
	Created  bool   `json:"created"`
}

type SceneOpenResult struct {
	Path   string `json:"path,omitempty"`
	Root   string `json:"root,omitempty"`
	Opened bool   `json:"opened,omitempty"`
	Queued bool   `json:"queued,omitempty"`
	JobID  string `json:"job_id,omitempty"`
}

type SceneInstanceResult struct {
	Path      string `json:"path"`
	Scene     string `json:"scene"`
	Parent    string `json:"parent"`
	Name      string `json:"name"`
	Instanced bool   `json:"instanced"`
}

type SceneApplyResult struct {
	Root    string `json:"root"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	DryRun  bool   `json:"dry_run,omitempty"`
}

type NodePropertyResult struct {
	Path     string `json:"path"`
	Property string `json:"property"`
	Value    any    `json:"value,omitempty"`
}

type NodeSetResourceResult struct {
	Path     string `json:"path"`
	Property string `json:"property"`
	Resource string `json:"resource"`
	Set      bool   `json:"set"`
}

type NodeAttachScriptResult struct {
	Path     string `json:"path"`
	Script   string `json:"script"`
	Attached bool   `json:"attached"`
}

type ScriptCheckResult struct {
	Path  string `json:"path"`
	Valid bool   `json:"valid"`
}

type ScriptCreateResult struct {
	Path    string `json:"path"`
	Valid   bool   `json:"valid"`
	Created bool   `json:"created"`
}

type ScriptWriteResult struct {
	Path    string `json:"path"`
	Valid   bool   `json:"valid"`
	Written bool   `json:"written"`
}

type ShaderCheckResult struct {
	Path  string `json:"path"`
	Valid bool   `json:"valid"`
}

type ShaderWriteResult struct {
	Path    string `json:"path"`
	Valid   bool   `json:"valid"`
	Written bool   `json:"written"`
}

type ResourceCreateResult struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Script  string `json:"script,omitempty"`
	Created bool   `json:"created"`
}

type AutoloadResult struct {
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"`
	Key     string `json:"key,omitempty"`
	Added   bool   `json:"added,omitempty"`
	Removed bool   `json:"removed,omitempty"`
}

type AutoloadListResult struct {
	Autoloads []AutoloadResult `json:"autoloads"`
}

type InputEventInfo struct {
	Type     string `json:"type"`
	Keycode  int    `json:"keycode,omitempty"`
	Key      string `json:"key,omitempty"`
	Physical bool   `json:"physical,omitempty"`
	Text     string `json:"text,omitempty"`
}

type InputActionResult struct {
	Action     string           `json:"action"`
	Deadzone   float64          `json:"deadzone,omitempty"`
	Events     []InputEventInfo `json:"events,omitempty"`
	Project    bool             `json:"project,omitempty"`
	Added      bool             `json:"added,omitempty"`
	Removed    bool             `json:"removed,omitempty"`
	EventAdded bool             `json:"event_added,omitempty"`
	Key        string           `json:"key,omitempty"`
	Physical   bool             `json:"physical,omitempty"`
}

type InputActionListResult struct {
	Actions []InputActionResult `json:"actions"`
}

type FileWriteBytesResult struct {
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
	Written bool   `json:"written"`
}

type ViewportScreenshotResult struct {
	Format        string `json:"format,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Index         int    `json:"index,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	ContentBase64 string `json:"content_base64,omitempty"`
	Queued        bool   `json:"queued,omitempty"`
	JobID         string `json:"job_id,omitempty"`
}

type NodeGroupResult struct {
	Path    string `json:"path"`
	Group   string `json:"group"`
	Added   bool   `json:"added,omitempty"`
	Removed bool   `json:"removed,omitempty"`
}

type NodeGroupListResult struct {
	Path   string   `json:"path"`
	Groups []string `json:"groups"`
}

type SignalConnectResult struct {
	From      string `json:"from"`
	Signal    string `json:"signal"`
	To        string `json:"to"`
	Method    string `json:"method"`
	Connected bool   `json:"connected"`
}

type SignalDisconnectResult struct {
	From         string `json:"from"`
	Signal       string `json:"signal"`
	To           string `json:"to"`
	Method       string `json:"method"`
	Disconnected bool   `json:"disconnected"`
}

type ProjectSettingResult struct {
	Key   string `json:"key"`
	Value any    `json:"value,omitempty"`
	Set   bool   `json:"set,omitempty"`
}

type NodeDuplicateResult struct {
	SourcePath string `json:"source_path"`
	Path       string `json:"path"`
	DryRun     bool   `json:"dry_run,omitempty"`
	Duplicated bool   `json:"duplicated,omitempty"`
}

type PropertyInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Usage int    `json:"usage"`
}

type NodeListPropertiesResult struct {
	Path       string         `json:"path"`
	Properties []PropertyInfo `json:"properties"`
}

type FileListResult struct {
	Path  string   `json:"path"`
	Files []string `json:"files"`
	Dirs  []string `json:"dirs"`
}

type FileMkdirResult struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
}

type FileDeleteResult struct {
	Path    string `json:"path"`
	Deleted bool   `json:"deleted"`
}

type FileExistsResult struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	IsFile bool   `json:"is_file"`
	IsDir  bool   `json:"is_dir"`
}

type NavigationBakeResult struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Baked bool   `json:"baked"`
}

type ImportSetResult struct {
	Path    string `json:"path"`
	Params  int    `json:"params"`
	Applied bool   `json:"applied"`
}

type SceneListResult struct {
	Dir    string   `json:"dir"`
	Scenes []string `json:"scenes"`
}

type ResourceListResult struct {
	Dir       string   `json:"dir"`
	Resources []string `json:"resources"`
}

type LogEntry struct {
	Time    string         `json:"time"`
	Level   string         `json:"level"`
	Source  string         `json:"source"`
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

type LogsResponse struct {
	OK      bool       `json:"ok"`
	Entries []LogEntry `json:"entries"`
}

type RunStartResult struct {
	Running      bool   `json:"running"`
	Scene        string `json:"scene,omitempty"`
	PlayingScene string `json:"playing_scene,omitempty"`
}

type RunStatusResult struct {
	Running      bool          `json:"running"`
	PlayingScene string        `json:"playing_scene,omitempty"`
	Debugger     DebuggerState `json:"debugger,omitempty"`
}

type RunStopResult struct {
	Stopped      bool   `json:"stopped"`
	Running      bool   `json:"running"`
	PlayingScene string `json:"playing_scene,omitempty"`
}

type RunScreenshotResult struct {
	Format        string `json:"format,omitempty"`
	Source        string `json:"source,omitempty"`
	Screen        int    `json:"screen,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	ContentBase64 string `json:"content_base64,omitempty"`
	Queued        bool   `json:"queued,omitempty"`
	JobID         string `json:"job_id,omitempty"`
}

type RunInputResult struct {
	Queued     bool   `json:"queued,omitempty"`
	JobID      string `json:"job_id,omitempty"`
	Steps      int    `json:"steps,omitempty"`
	DurationMS int    `json:"duration_ms,omitempty"`
}

type DebuggerFrame struct {
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

type DebuggerState struct {
	Paused      bool             `json:"paused"`
	Reason      string           `json:"reason,omitempty"`
	Message     string           `json:"message,omitempty"`
	File        string           `json:"file,omitempty"`
	Line        int              `json:"line,omitempty"`
	Function    string           `json:"function,omitempty"`
	Stack       []map[string]any `json:"stack,omitempty"`
	StackFrames []DebuggerFrame  `json:"stack_frames,omitempty"`
	RawData     map[string]any   `json:"raw_data,omitempty"`
	UpdatedAt   string           `json:"updated_at,omitempty"`
}

type Job struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Status    string         `json:"status"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Detail    map[string]any `json:"detail,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
	Error     *BridgeError   `json:"error,omitempty"`
}

type JobResponse struct {
	OK  bool `json:"ok"`
	Job Job  `json:"job"`
}

// Phase 4 types

type RunRaycastResult struct {
	Queued       bool      `json:"queued,omitempty"`
	JobID        string    `json:"job_id,omitempty"`
	CameraPath   string    `json:"camera_path,omitempty"`
	RayOrigin    []float64 `json:"ray_origin,omitempty"`
	RayDirection []float64 `json:"ray_direction,omitempty"`
	HitCollider  string    `json:"hit_collider,omitempty"`
	HitPosition  []float64 `json:"hit_position,omitempty"`
	HitNormal    []float64 `json:"hit_normal,omitempty"`
	HitDistance  float64   `json:"hit_distance,omitempty"`
	Hit          bool      `json:"hit"`
}

type SceneApplyBlueprintResult struct {
	Path      string `json:"path,omitempty"`
	Blueprint string `json:"blueprint,omitempty"`
	Created   int    `json:"created,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
}

type ThemeCreateResult struct {
	Path    string `json:"path,omitempty"`
	Created bool   `json:"created,omitempty"`
}

type ThemeSetResult struct {
	Path     string `json:"path,omitempty"`
	NodeType string `json:"node_type,omitempty"`
	DataType string `json:"data_type,omitempty"`
	Name     string `json:"name,omitempty"`
	Set      bool   `json:"set,omitempty"`
}

type AnimationCreateResult struct {
	Path    string `json:"path,omitempty"`
	Name    string `json:"name,omitempty"`
	Created bool   `json:"created,omitempty"`
}

type AnimationTrackResult struct {
	Path      string `json:"path,omitempty"`
	Animation string `json:"animation,omitempty"`
	TrackIdx  int    `json:"track_idx"`
}

type AnimationKeyframeResult struct {
	Path      string  `json:"path,omitempty"`
	Animation string  `json:"animation,omitempty"`
	TrackIdx  int     `json:"track_idx"`
	Time      float64 `json:"time"`
	Added     bool    `json:"added,omitempty"`
}

type TilesetCreateResult struct {
	Path    string `json:"path,omitempty"`
	Created bool   `json:"created,omitempty"`
}

type TilemapCellResult struct {
	Node    string `json:"node,omitempty"`
	Layer   int    `json:"layer"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Applied bool   `json:"applied,omitempty"`
}

type AudioBusResult struct {
	Bus     string `json:"bus,omitempty"`
	Applied bool   `json:"applied,omitempty"`
}

type ViewportSetSizeResult struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Path   string `json:"path,omitempty"`
}

type ViewportAddResult struct {
	Path   string `json:"path,omitempty"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Added  bool   `json:"added,omitempty"`
}
