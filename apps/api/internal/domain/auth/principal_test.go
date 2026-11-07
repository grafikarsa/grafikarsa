package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestComputeNextStep(t *testing.T) {
	activated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name         string
		activatedAt  *time.Time
		passwordKind string
		want         NextStep
	}{
		{name: "not activated", activatedAt: nil, passwordKind: "initial", want: NextStepActivate},
		{name: "temporary password", activatedAt: &activated, passwordKind: "temporary", want: NextStepChangePassword},
		{name: "user password", activatedAt: &activated, passwordKind: "user", want: NextStepNone},
		{name: "initial password after activation", activatedAt: &activated, passwordKind: "initial", want: NextStepNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeNextStep(tt.activatedAt, tt.passwordKind); got != tt.want {
				t.Fatalf("ComputeNextStep() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeGrants(t *testing.T) {
	majorA := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	majorB := uuid.MustParse("00000000-0000-0000-0000-00000000000b")

	tests := []struct {
		name     string
		explicit []grantRow
		implicit []string
		want     []Grant
	}{
		{
			name:     "implicit permissions are global",
			implicit: []string{"content.create"},
			want: []Grant{
				{Permission: "content.create", Global: true},
			},
		},
		{
			name: "explicit scoped grant keeps majors sorted and unique",
			explicit: []grantRow{
				{Permission: "content.moderate", MajorID: &majorB},
				{Permission: "content.moderate", MajorID: &majorA},
				{Permission: "content.moderate", MajorID: &majorB},
			},
			want: []Grant{
				{Permission: "content.moderate", MajorIDs: []uuid.UUID{majorA, majorB}},
			},
		},
		{
			name: "global explicit grant collapses scopes",
			explicit: []grantRow{
				{Permission: "content.moderate", MajorID: &majorA},
				{Permission: "content.moderate"},
			},
			implicit: []string{"content.create"},
			want: []Grant{
				{Permission: "content.create", Global: true},
				{Permission: "content.moderate", Global: true},
			},
		},
		{
			name: "implicit overrides scoped grant",
			explicit: []grantRow{
				{Permission: "content.create", MajorID: &majorA},
			},
			implicit: []string{"content.create"},
			want: []Grant{
				{Permission: "content.create", Global: true},
			},
		},
		{
			name: "results are sorted by permission",
			explicit: []grantRow{
				{Permission: "user.view", MajorID: &majorA},
				{Permission: "audit.view"},
			},
			implicit: []string{"content.create"},
			want: []Grant{
				{Permission: "audit.view", Global: true},
				{Permission: "content.create", Global: true},
				{Permission: "user.view", MajorIDs: []uuid.UUID{majorA}},
			},
		},
		{name: "empty input", want: []Grant{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeGrants(tt.explicit, tt.implicit)
			if len(got) != len(tt.want) {
				t.Fatalf("mergeGrants() has %d grants, want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				assertGrant(t, i, got[i], tt.want[i])
			}
		})
	}
}

func assertGrant(t *testing.T, index int, got, want Grant) {
	t.Helper()

	if got.Permission != want.Permission || got.Global != want.Global {
		t.Fatalf("grant %d = %+v, want %+v", index, got, want)
	}
	if len(got.MajorIDs) != len(want.MajorIDs) {
		t.Fatalf("grant %d majors = %v, want %v", index, got.MajorIDs, want.MajorIDs)
	}
	for i := range want.MajorIDs {
		if got.MajorIDs[i] != want.MajorIDs[i] {
			t.Fatalf("grant %d major %d = %s, want %s", index, i, got.MajorIDs[i], want.MajorIDs[i])
		}
	}
}

func TestPrincipalPermissionChecks(t *testing.T) {
	majorA := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	majorB := uuid.MustParse("00000000-0000-0000-0000-00000000000b")

	principal := &Principal{
		Grants: []Grant{
			{Permission: "content.create", Global: true},
			{Permission: "content.moderate", MajorIDs: []uuid.UUID{majorA}},
			{Permission: "user.view", Global: false, MajorIDs: []uuid.UUID{majorB}},
		},
	}

	tests := []struct {
		name       string
		permission string
		wantCan    bool
		wantGlobal bool
		major      uuid.UUID
		wantMajor  bool
	}{
		{
			name:       "global grant",
			permission: "content.create",
			wantCan:    true,
			wantGlobal: true,
			major:      majorB,
			wantMajor:  true,
		},
		{
			name:       "scoped grant matches its major",
			permission: "content.moderate",
			wantCan:    true,
			wantGlobal: false,
			major:      majorA,
			wantMajor:  true,
		},
		{
			name:       "scoped grant does not match another major",
			permission: "content.moderate",
			wantCan:    true,
			wantGlobal: false,
			major:      majorB,
			wantMajor:  false,
		},
		{
			name:       "scoped grant matches only its own major",
			permission: "user.view",
			wantCan:    true,
			wantGlobal: false,
			major:      majorA,
			wantMajor:  false,
		},
		{
			name:       "unknown permission",
			permission: "settings.manage",
			wantCan:    false,
			wantGlobal: false,
			major:      majorA,
			wantMajor:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := principal.Can(tt.permission); got != tt.wantCan {
				t.Errorf("Can(%q) = %v, want %v", tt.permission, got, tt.wantCan)
			}
			if got := principal.CanGlobally(tt.permission); got != tt.wantGlobal {
				t.Errorf("CanGlobally(%q) = %v, want %v", tt.permission, got, tt.wantGlobal)
			}
			if got := principal.CanForMajor(tt.permission, tt.major); got != tt.wantMajor {
				t.Errorf("CanForMajor(%q, %s) = %v, want %v", tt.permission, tt.major, got, tt.wantMajor)
			}
		})
	}
}
