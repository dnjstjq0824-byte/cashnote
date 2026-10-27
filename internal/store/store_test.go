package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestStoreService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	bizID := uuid.New()
	ownerID := uuid.New()
	managerID := uuid.New()
	staffID := uuid.New()
	outsiderID := uuid.New()

	t.Run("Create store automatically assigns OWNER", func(t *testing.T) {
		st, err := svc.CreateStore(ctx, bizID, ownerID, "강남본점", "F&B", "서울시 강남구 테헤란로 123")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}

		if st.StoreName != "강남본점" {
			t.Errorf("expected store name 강남본점, got %s", st.StoreName)
		}

		// Owner check
		hasPerm, err := svc.CheckPermission(ctx, st.ID, ownerID, RoleOwner)
		if err != nil || !hasPerm {
			t.Fatalf("owner should have RoleOwner permission")
		}

		// Add manager & staff
		_, err = svc.AddMember(ctx, st.ID, managerID, RoleManager)
		if err != nil {
			t.Fatalf("failed to add manager: %v", err)
		}
		_, err = svc.AddMember(ctx, st.ID, staffID, RoleStaff)
		if err != nil {
			t.Fatalf("failed to add staff: %v", err)
		}

		// Check manager has RoleManager and RoleStaff permission
		ok, err := svc.CheckPermission(ctx, st.ID, managerID, RoleStaff)
		if err != nil || !ok {
			t.Errorf("manager should have staff level permission")
		}

		// Staff does NOT have RoleOwner or RoleManager permission
		_, err = svc.CheckPermission(ctx, st.ID, staffID, RoleManager)
		if err == nil {
			t.Errorf("staff should be forbidden from manager operations")
		}

		// Outsider is forbidden
		_, err = svc.CheckPermission(ctx, st.ID, outsiderID, RoleStaff)
		if err == nil {
			t.Errorf("outsider should be forbidden from store access")
		}
	})
}
