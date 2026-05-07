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

type NodePropertyResult struct {
	Path     string `json:"path"`
	Property string `json:"property"`
	Value    any    `json:"value,omitempty"`
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
