package editor

// Editor 描述一个系统上可用的文本编辑器。
type Editor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}
