package seed

import "messenger-backend/internal/domain/entity"

type AdminChatConfig struct {
	Platform       string `json:"platform"`
	PlatformChatID string `json:"platform_chat_id"`
	Title          string `json:"title"`
	Username       string `json:"username"`
	ChatType       string `json:"chat_type"`
	IsPrivate      bool   `json:"is_private"`
}

type Config struct {
	CompanyName               string
	CompanyCode               string
	SuperAdminUsername        string
	SuperAdminPassword        string
	SuperAdminEmail           string
	SuperAdminPhone           string
	SuperAdminChatsRawPayload string // json payload
}

type policyRule struct {
	Role       string
	Domain     string
	Permission entity.Permission
}

// DefaultAdminPolicies contains the static RBAC permission set for admin seeding.
var DefaultAdminPolicies = []policyRule{

	// Broadcast
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/batch", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id", Method: "PUT"}},

	// Chats & Chat Histories
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/otp/send", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/otp/send", Method: "POST"}},

	// Companies
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/me", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/users/send-register-otp", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/:id", Method: "DELETE"}},

	// Users
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/me", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "GET"}},
	// {Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users", Method: "POST"}}, ?!?!?
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/email", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/phone", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/username", Method: "PUT"}},

	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/roles", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "DELETE"}},

	// Attachments
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/by-messages", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/by-platform-file/:platform_file_id", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/batch", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/thumbnail", Method: "PATCH"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/restore", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/links/download", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/links/download/batch", Method: "POST"}},

	// Message Attachments (Nested)
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/batch", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/links/download", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/links/download/batch", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "DELETE"}},
}
