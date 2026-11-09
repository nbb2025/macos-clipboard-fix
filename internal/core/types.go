package core

type Check struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}
type ServiceState struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	PID     int    `json:"pid"`
	Running bool   `json:"running"`
}
type CacheState struct {
	Kind       string `json:"kind"`
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	Bytes      int64  `json:"bytes"`
	Modified   string `json:"modified"`
	AgeSeconds int64  `json:"ageSeconds"`
	Detail     string `json:"detail"`
}
type ToolState struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
}
type ProxyState struct {
	Available  bool   `json:"available"`
	TUN        bool   `json:"tun"`
	Mode       string `json:"mode"`
	Stack      string `json:"stack"`
	AppleRoute string `json:"appleRoute"`
}
type Signal struct {
	Title  string `json:"title"`
	Count  int    `json:"count"`
	Detail string `json:"detail"`
}
type Report struct {
	At              string         `json:"at"`
	Version         string         `json:"version"`
	Model           string         `json:"model"`
	Checks          []Check        `json:"checks"`
	Services        []ServiceState `json:"services"`
	Caches          []CacheState   `json:"caches"`
	Tools           []ToolState    `json:"tools"`
	Proxy           ProxyState     `json:"proxy"`
	Findings        []string       `json:"findings"`
	SuggestedAction string         `json:"suggestedAction"`
}
type Event struct {
	At      string `json:"at"`
	Level   string `json:"level"`
	Message string `json:"message"`
}
type Progress struct {
	Action  string `json:"action"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
	State   string `json:"state"`
}
type Backup struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Original string `json:"original"`
	Path     string `json:"path"`
	Created  string `json:"created"`
	Bytes    int64  `json:"bytes"`
	Restored bool   `json:"restored"`
	Ready    bool   `json:"ready"`
}
type Probe struct {
	Text    string `json:"text"`
	Created string `json:"created"`
	Matches bool   `json:"matches"`
}
type Verification struct {
	Direction string `json:"direction"`
	Success   bool   `json:"success"`
	At        string `json:"at"`
}
type Session struct {
	History       []Event        `json:"history"`
	Backups       []Backup       `json:"backups"`
	Verifications []Verification `json:"verifications"`
}
