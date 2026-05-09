package bridge

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
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
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

type MaterialWriteResult struct {
	Path          string            `json:"path"`
	Shader        string            `json:"shader"`
	TextureParams map[string]string `json:"texture_params,omitempty"`
	Written       bool              `json:"written"`
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
