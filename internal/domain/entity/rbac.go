package entity

type RolePermissions struct {
	Role        string       `json:"role"`
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	Endpoint string `json:"endpoint"` // obj
	Method   string `json:"method"`   // act
}

type UserRoleMapping struct {
	User  string   `json:"user"`
	Roles []string `json:"roles"`
}
