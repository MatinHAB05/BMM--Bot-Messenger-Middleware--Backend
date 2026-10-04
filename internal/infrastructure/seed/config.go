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

	// ==========================================
	// 1. Shared Permissions (SuperAdmin & Admin)
	// ==========================================

	// Broadcast (Full access for both)
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/batch", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id/platforms", Method: "GET"}},

	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/batch", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast/:id/platforms", Method: "GET"}},

	// Chats & Chat Histories (Full access for both)
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "GET"}},
	// {Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "POST"}},  it dose not make sense by our bussines logic
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "DELETE"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/otp/send", Method: "POST"}},

	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/otp/send", Method: "POST"}},

	// Attachments (Full access for both)
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

	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/by-messages", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/by-platform-file/:platform_file_id", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/batch", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/thumbnail", Method: "PATCH"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/restore", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/links/download", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/attachments/:id/links/download/batch", Method: "POST"}},

	// Message Attachments (Full access for both)
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/batch", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/links/download", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/links/download/batch", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "DELETE"}},

	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/batch", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/links/download", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments/links/download/batch", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id/attachments", Method: "DELETE"}},

	// Self Profiles (Both roles)
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/me", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/me", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/me", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/me", Method: "GET"}},

	// ==========================================
	// 2. Exclusive SuperAdmin Permissions
	// ==========================================

	// Company Administration
	// {Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies", Method: "POST"}}, it dose not make sense by our bussines logic
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/users/send-register-otp", Method: "POST"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/:id", Method: "DELETE"}},

	// User Management Lifecycle
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users", Method: "GET"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "GET"}},
	// {Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users", Method: "POST"}},  it dose not make sense by our bussines logic
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/email", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/phone", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/username", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/roles", Method: "PUT"}},
	{Role: entity.RoleSuperAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "DELETE"}},
}
