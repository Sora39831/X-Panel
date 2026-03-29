package service

import (
	"path/filepath"
	"testing"

	"x-ui/database"
	"x-ui/database/model"
)

func TestUpdateFirstUser_ForceAdminRole(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "x-ui-test.db")
	if err := database.InitProvider("sqlite", dbPath); err != nil {
		t.Fatalf("init provider failed: %v", err)
	}

	db := database.GetDB()
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	var first model.User
	if err := db.First(&first).Error; err != nil {
		t.Fatalf("get first user failed: %v", err)
	}

	if err := db.Model(&model.User{}).Where("id = ?", first.Id).Update("role", "user").Error; err != nil {
		t.Fatalf("prepare non-admin role failed: %v", err)
	}

	svc := UserService{}
	if err := svc.UpdateFirstUser("installer-admin", "installer-pass"); err != nil {
		t.Fatalf("update first user failed: %v", err)
	}

	var updated model.User
	if err := db.First(&updated).Error; err != nil {
		t.Fatalf("reload first user failed: %v", err)
	}

	if updated.Role != "admin" {
		t.Fatalf("expected role admin, got %q", updated.Role)
	}
}

