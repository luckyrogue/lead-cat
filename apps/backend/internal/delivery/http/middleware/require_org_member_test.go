package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres" //nolint:depguard
)

type fakeOrgResolver struct {
	member postgres.Member
	ok     bool
}

func (f fakeOrgResolver) GetOrgMember(_ context.Context, _, _ uuid.UUID) (postgres.Member, bool, error) {
	return f.member, f.ok, nil
}

func appWithWebUser(handler fiber.Handler, mws ...fiber.Handler) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("web_user", postgres.PlatformUser{ID: uuid.New()})
		return c.Next()
	})
	chain := append(mws, handler)
	app.Get("/orgs/:id/x", chain...)
	app.Patch("/orgs/:id/x", chain...)
	return app
}

func okHandler(c *fiber.Ctx) error { return c.SendString("ok") }

func TestRequireOrgMemberRejectsNonMember(t *testing.T) {
	app := appWithWebUser(okHandler, RequireOrgMember(fakeOrgResolver{ok: false}))
	req := httptest.NewRequest(http.MethodGet, "/orgs/"+uuid.NewString()+"/x", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}

func TestRequireOrgMemberAllowsMember(t *testing.T) {
	app := appWithWebUser(okHandler, RequireOrgMember(fakeOrgResolver{ok: true, member: postgres.Member{Role: "member"}}))
	req := httptest.NewRequest(http.MethodGet, "/orgs/"+uuid.NewString()+"/x", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestRequireOrgMemberBadOrgID(t *testing.T) {
	app := appWithWebUser(okHandler, RequireOrgMember(fakeOrgResolver{ok: true}))
	req := httptest.NewRequest(http.MethodGet, "/orgs/not-a-uuid/x", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

func TestRequireOrgRoleBlocksInsufficient(t *testing.T) {
	res := fakeOrgResolver{ok: true, member: postgres.Member{Role: "member"}}
	app := appWithWebUser(okHandler, RequireOrgMember(res), RequireOrgRole("admin"))
	req := httptest.NewRequest(http.MethodPatch, "/orgs/"+uuid.NewString()+"/x", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}

func TestRequireOrgRoleAllowsSufficient(t *testing.T) {
	res := fakeOrgResolver{ok: true, member: postgres.Member{Role: "admin"}}
	app := appWithWebUser(okHandler, RequireOrgMember(res), RequireOrgRole("admin"))
	req := httptest.NewRequest(http.MethodPatch, "/orgs/"+uuid.NewString()+"/x", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}
