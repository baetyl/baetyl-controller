package v3

type ListOptions struct {
	LabelSelector string `form:"selector,omitempty" json:"selector,omitempty"`
	NodeSelector  string `form:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`
	FieldSelector string `form:"fieldSelector,omitempty" json:"fieldSelector,omitempty"`
	Limit         int64  `form:"limit,omitempty" json:"limit,omitempty"`
	Continue      string `form:"continue,omitempty" json:"continue,omitempty"`
}
