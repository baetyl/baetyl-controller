package v3

import "time"

type Template struct {
	User        `json:",inline"`
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Version     string            `json:"version,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	Description string            `json:"description,omitempty"`
	CreateTime  time.Time         `json:"createTime,omitempty"`
}

type TemplateList struct {
	*ListOptions `json:",inline"`
	Total        int        `json:"total"`
	Items        []Template `json:"items"`
}
