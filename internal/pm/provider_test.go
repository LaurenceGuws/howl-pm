package pm

import "testing"

func TestProvidersRegistersAndroidUserlandAndTermuxMain(t *testing.T) {
	got := Providers("android")
	if len(got) != 2 {
		t.Fatalf("expected 2 android providers, got %d", len(got))
	}
	if got[0].ID != "android-userland" || got[1].ID != "termux-main" {
		t.Fatalf("unexpected provider ids: %#v", got)
	}
	if got[0].Upstream != "termux-main" || got[0].Scope != "subset" {
		t.Fatalf("android-userland must be termux-main subset, got %#v", got[0])
	}
}

func TestPublicProvidersExcludesInternalDefaults(t *testing.T) {
	got := PublicProviders("android")
	if len(got) != 1 {
		t.Fatalf("expected 1 public provider, got %d", len(got))
	}
	if got[0].ID != "termux-main" {
		t.Fatalf("expected termux-main, got %#v", got)
	}
	for _, p := range got {
		if p.ID == DevBaselinePackage {
			t.Fatalf("default package profile must not appear as provider: %#v", p)
		}
	}
}
