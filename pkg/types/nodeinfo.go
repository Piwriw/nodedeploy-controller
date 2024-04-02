package types

type NodeInfo struct {
	NodeName string `json:"nodeName,omitempty"`
	NodeIP   string `json:"nodeIP,omitempty"`
	NodeType string `json:"nodeType,omitempty"`
	NodePort string `json:"nodePort,omitempty"`
	NodeUser string `json:"nodeUser,omitempty"`
	NodePwd  string `json:"nodePwd,omitempty"`

	HarborEndpoint string `json:"harborEndpoint,omitempty"`
	HarborUser     string `json:"harborUser,omitempty"`
	HarborPwd      string `json:"harborPwd,omitempty"`
}
