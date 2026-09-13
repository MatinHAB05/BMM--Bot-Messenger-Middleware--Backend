// Package rbac wires Casbin's enforcer to PostgreSQL via the official GORM
// adapter, which owns and auto-migrates its own casbin_rule table
// (policies and role/grouping assignments both live there -- there is no
// separate "roles" table anywhere in this codebase).
package rbac

import (
	"fmt"
	"messenger-backend/internal/infrastructure/database"
	"sync"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

type Config struct {
	ModelConfigFilePath string
}

type RBACEnforcer interface {
	GetEnforcer() *casbin.Enforcer
	Enforce(rvals ...any) (bool, error)
	GetRBACConfig() Config
}

type Casbin struct {
	enforcer *casbin.Enforcer
	*Config
}

var (
	casbinOnce     sync.Once
	casbinInstance *Casbin
	casbinErr      error
)

func NewEnforcer(gormDB database.Database, casbinConfig *Config) (RBACEnforcer, error) {
	casbinOnce.Do(func() {
		adapter, err := gormadapter.NewAdapterByDB(gormDB.GetGormDB())
		if err != nil {
			casbinErr = fmt.Errorf("failed to create Casbin adapter: %w", err)
			return
		}

		m, err := model.NewModelFromFile(casbinConfig.ModelConfigFilePath)
		if err != nil {
			casbinErr = fmt.Errorf("failed to parse Casbin model configuration: %w", err)
			return
		}

		enforcer, err := casbin.NewEnforcer(m, adapter)
		if err != nil {
			casbinErr = fmt.Errorf("failed to create Casbin enforcer: %w", err)
			return
		}

		err = enforcer.LoadPolicy()
		if err != nil {
			casbinErr = fmt.Errorf("failed to load Casbin policies from DB: %w", err)
			return
		}

		casbinInstance = &Casbin{enforcer: enforcer, Config: casbinConfig}
	})

	if casbinErr != nil {
		return nil, casbinErr
	}

	return casbinInstance, nil
}

// GetEnforcer returns the underlying Casbin enforcer instance.
func (c *Casbin) GetEnforcer() *casbin.Enforcer {
	return c.enforcer
}

// Enforce evaluates raw arguments against the Casbin model.
func (c *Casbin) Enforce(rvals ...any) (bool, error) {
	return c.enforcer.Enforce(rvals...)
}

func (c *Casbin) GetRBACConfig() Config {
	return *c.Config
}
