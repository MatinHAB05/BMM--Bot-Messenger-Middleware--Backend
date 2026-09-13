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
	CompanyName          string
	CompanyCode          string
	AdminUsername        string
	AdminPassword        string
	AdminEmail           string
	AdminPhone           string
	AdminChatsRawPayload string // json payload
}

type policyRule struct {
	Role       string
	Domain     string
	Permission entity.Permission
}

// DefaultAdminPolicies contains the static RBAC permission set for admin seeding.
var DefaultAdminPolicies = []policyRule{
	// Broadcast
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/broadcast", Method: "POST"}},

	// Chats & Chat Histories
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/:id/history/:message_id", Method: "DELETE"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/chats/otp/send", Method: "POST"}},

	// Companies
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/me", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/users/send-register-otp", Method: "POST"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/:id", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/companies/:id", Method: "DELETE"}},

	// Users
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/me", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users", Method: "GET"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "GET"}},
	// {Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users", Method: "POST"}}, ?!?!?
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/email", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/phone", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/username", Method: "PUT"}},

	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id/roles", Method: "PUT"}},
	{Role: entity.RoleAdmin, Domain: "*", Permission: entity.Permission{Endpoint: "/api/v1/users/:id", Method: "DELETE"}},
}
