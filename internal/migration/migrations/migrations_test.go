package migrations

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"monera-digital/internal/migration"
)

// Interface/Version/Description tests for the new C-2 migration 046.
// The full SQL behaviour is exercised by running `go run ./cmd/migrate`
// against a real (or test) database; sqlmock coverage of every
// statement would duplicate the integration value without catching
// additional regressions.

func TestAddPendingStatusAndActivationFields_Interface(t *testing.T) {
	var _ migration.Migration = (*AddPendingStatusAndActivationFields)(nil)
}

func TestAddPendingStatusAndActivationFields_Version(t *testing.T) {
	m := &AddPendingStatusAndActivationFields{}
	if v := m.Version(); v != "046" {
		t.Errorf("Version() = %q, want %q", v, "046")
	}
}

func TestAddPendingStatusAndActivationFields_Description(t *testing.T) {
	m := &AddPendingStatusAndActivationFields{}
	if m.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestNormalizeAmountTypes_Interface(t *testing.T) {
	var _ migration.Migration = (*NormalizeAmountTypes)(nil)
}

func TestNormalizeAmountTypes_Version(t *testing.T) {
	m := &NormalizeAmountTypes{}
	if v := m.Version(); v != "047" {
		t.Errorf("Version() = %q, want %q", v, "047")
	}
}

func TestNormalizeAmountTypes_Description(t *testing.T) {
	m := &NormalizeAmountTypes{}
	if m.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestAddMissingForeignKeys_Interface(t *testing.T) {
	var _ migration.Migration = (*AddMissingForeignKeys)(nil)
}

func TestAddMissingForeignKeys_Version(t *testing.T) {
	m := &AddMissingForeignKeys{}
	if v := m.Version(); v != "048" {
		t.Errorf("Version() = %q, want %q", v, "048")
	}
}

func TestAddMissingForeignKeys_Description(t *testing.T) {
	m := &AddMissingForeignKeys{}
	if m.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestWidenAmountPrecision_InterfaceAndVersion(t *testing.T) {
	var _ migration.Migration = (*WidenAmountPrecision)(nil)
	if version := (&WidenAmountPrecision{}).Version(); version != "051" {
		t.Fatalf("Version() = %q, want 051", version)
	}
}

func TestExpandCompanyFundOccurrenceAndManualValuationInterfaceAndVersion(t *testing.T) {
	var _ migration.Migration = (*ExpandCompanyFundOccurrenceAndManualValuation)(nil)
	if version := (&ExpandCompanyFundOccurrenceAndManualValuation{}).Version(); version != "052" {
		t.Fatalf("Version() = %q, want 052", version)
	}
}

func TestEnforceSafeheronOccurrenceInterfaceAndVersion(t *testing.T) {
	var _ migration.Migration = (*EnforceSafeheronOccurrence)(nil)
	var _ migration.ControlledMigration = (*EnforceSafeheronOccurrence)(nil)
	if version := (&EnforceSafeheronOccurrence{}).Version(); version != "053" {
		t.Fatalf("Version() = %q, want 053", version)
	}
}

// TestAddTwoFactorColumnsMigration_Interface verifies the migration implements the interface
func TestAddTwoFactorColumnsMigration_Interface(t *testing.T) {
	var _ migration.Migration = (*AddTwoFactorColumnsMigration)(nil)
}

// TestAddTwoFactorColumnsMigration_Version verifies version
func TestAddTwoFactorColumnsMigration_Version(t *testing.T) {
	m := &AddTwoFactorColumnsMigration{}
	if m.Version() != "004" {
		t.Errorf("Expected version '004', got '%s'", m.Version())
	}
}

// TestAddTwoFactorColumnsMigration_Description verifies description
func TestAddTwoFactorColumnsMigration_Description(t *testing.T) {
	m := &AddTwoFactorColumnsMigration{}
	if m.Description() == "" {
		t.Error("Description should not be empty")
	}
}

// TestAddTwoFactorTimestampMigration_Interface verifies the migration implements the interface
func TestAddTwoFactorTimestampMigration_Interface(t *testing.T) {
	var _ migration.Migration = (*AddTwoFactorTimestampMigration)(nil)
}

// TestAddTwoFactorTimestampMigration_Version verifies version
func TestAddTwoFactorTimestampMigration_Version(t *testing.T) {
	m := &AddTwoFactorTimestampMigration{}
	if m.Version() != "005" {
		t.Errorf("Expected version '005', got '%s'", m.Version())
	}
}

// TestAddTwoFactorTimestampMigration_Description verifies description
func TestAddTwoFactorTimestampMigration_Description(t *testing.T) {
	m := &AddTwoFactorTimestampMigration{}
	if m.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestMigrationRunnerRegistersVersionsInOrder(t *testing.T) {
	cmd := exec.Command("go", "run", "../../../cmd/migrate", "-print-versions")
	cmd.Env = append(os.Environ(), "APP_ENV=production")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("print migration registry: %v\n%s", err, output)
	}
	var versions []string
	if err := json.Unmarshal(output, &versions); err != nil {
		t.Fatalf("decode migration registry: %v\n%s", err, output)
	}

	if len(versions) == 0 {
		t.Fatal("migration registry is empty")
	}
	for index, version := range versions {
		if index > 0 && version <= versions[index-1] {
			t.Fatalf("migration registry is not ordered: %v", versions)
		}
	}
	wantTail := []string{"046", "047", "048", "049", "050", "051", "052", "053", "054", "055", "056", "057", "058", "059", "060", "061", "062", "063", "064"}
	if len(versions) < len(wantTail) {
		t.Fatalf("migration registry is incomplete: %v", versions)
	}
	gotTail := versions[len(versions)-len(wantTail):]
	for index := range wantTail {
		if gotTail[index] != wantTail[index] {
			t.Fatalf("migration registry tail = %v, want %v", gotTail, wantTail)
		}
	}
}
