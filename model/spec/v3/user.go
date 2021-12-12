package v3

type User struct {
	UserName         string `json:"userName,omitempty"`
	UserId           string `json:"userId,omitempty"`
	OrganizationName string `json:"organizationName,omitempty"`
	OrganizationId   string `json:"organizationId,omitempty"`
	ProjectName      string `json:"projectName,omitempty"`
	ProjectId        string `json:"projectId,omitempty"`
}
