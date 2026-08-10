// cmd/migrate/main.go
//
// MoneraDigital Go migration runner. Replaces the previous dead state where
// the binary was excluded via //go:build ignore. Run from repo root:
//
//   MIGRATION_DATABASE_URL=... go run ./cmd/migrate       # preferred on stage/prod (direct)
//   DATABASE_URL=... go run ./cmd/migrate                 # local fallback if direct
//   ... go run ./cmd/migrate -dry-run
//   ... go run ./cmd/migrate -rollback
//   EXPECTED_MIGRATION_CEILING=050 MIGRATION_DATABASE_URL=... \
//     go run ./cmd/migrate -exact-version 050
//
// Stage/production require MIGRATION_DATABASE_URL (direct/unpooled). Pooler
// hosts are rejected. Advisory lock wait is bounded (default 30s).
// See docs/security/MIGRATION-NOTES.md and ADR 0003.

package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"monera-digital/internal/buildinfo"
	"monera-digital/internal/migration"
	"monera-digital/internal/migration/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

var version = "dev"

var artifactMigrationReleaseSequence = []string{"063"}

type migrationDescriptor struct {
	version          string
	predecessor      string
	exactDeploy      bool
	artifactCeiling  bool
	newMigrationFunc func() migration.Migration
}

var migrationRegistry = []migrationDescriptor{
	{version: "001", newMigrationFunc: func() migration.Migration { return &migrations.CreateUsersTable{} }},
	{version: "002", newMigrationFunc: func() migration.Migration { return &migrations.CreateLendingPositionsTable{} }},
	{version: "003", newMigrationFunc: func() migration.Migration { return &migrations.CreateWithdrawalTables{} }},
	{version: "004", newMigrationFunc: func() migration.Migration { return &migrations.AddTwoFactorColumnsMigration{} }},
	{version: "005", newMigrationFunc: func() migration.Migration { return &migrations.AddTwoFactorTimestampMigration{} }},
	{version: "007", newMigrationFunc: func() migration.Migration { return &migrations.UpdateWalletRequestsTable{} }},
	{version: "008", newMigrationFunc: func() migration.Migration { return &migrations.CreateUserWalletsTable{} }},
	{version: "009", newMigrationFunc: func() migration.Migration { return &migrations.AddUserWalletStatusField{} }},
	{version: "010", newMigrationFunc: func() migration.Migration { return &migrations.AddIsPrimaryToWhitelist{} }},
	{version: "011", newMigrationFunc: func() migration.Migration { return &migrations.CreateDepositsTable{} }},
	{version: "012", newMigrationFunc: func() migration.Migration { return &migrations.AddUserStatus{} }},
	{version: "013", newMigrationFunc: func() migration.Migration { return &migrations.AddFrozenUntilToWhitelist{} }},
	{version: "014", newMigrationFunc: func() migration.Migration { return &migrations.AddEmailVerifiedStatusAndContactFields{} }},
	{version: "015", newMigrationFunc: func() migration.Migration { return &migrations.SafeheronPhase1{} }},
	{version: "016", newMigrationFunc: func() migration.Migration { return &migrations.AccountFrozenBalanceDefault{} }},
	{version: "046", newMigrationFunc: func() migration.Migration { return &migrations.AddPendingStatusAndActivationFields{} }},
	{version: "047", newMigrationFunc: func() migration.Migration { return &migrations.NormalizeAmountTypes{} }},
	{version: "048", newMigrationFunc: func() migration.Migration { return &migrations.AddMissingForeignKeys{} }},
	{version: "049", newMigrationFunc: func() migration.Migration { return &migrations.CreateFundReports{} }},
	{version: "050", predecessor: "049", exactDeploy: true, newMigrationFunc: func() migration.Migration { return &migrations.CreateCompanyFundLedger{} }},
	{version: "051", predecessor: "050", exactDeploy: true, newMigrationFunc: func() migration.Migration { return &migrations.WidenAmountPrecision{} }},
	{version: "052", predecessor: "051", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.ExpandCompanyFundOccurrenceAndManualValuation{} }},
	{version: "053", predecessor: "052", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.EnforceSafeheronOccurrence{} }},
	{version: "054", predecessor: "053", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.AllowManualCompanyFundTransactions{} }},
	{version: "055", predecessor: "054", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.AddCounterpartyNameOverride{} }},
	{version: "056", predecessor: "055", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.UnifySafeheronAddressOwnership{} }},
	{version: "057", predecessor: "056", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.CreateSafeheronRoutingCases{} }},
	{version: "058", predecessor: "057", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.ScopeSafeheronProviderEventsByOccurrence{} }},
	{version: "059", predecessor: "058", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.AllowOtherCompanyFundAccounts{} }},
	{version: "060", predecessor: "059", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.AddManualTransactionVoidColumns{} }},
	{version: "061", predecessor: "060", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.CreateWithdrawalsTable{} }},
	{version: "062", predecessor: "061", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration { return &migrations.AddAirwallexPhase2Ledger{} }},
	{version: "063", predecessor: "062", exactDeploy: true, artifactCeiling: true, newMigrationFunc: func() migration.Migration {
		return &migrations.AddAirwallexAccountLifecycle{
			LegacyMappingJSON: os.Getenv("AIRWALLEX_LEGACY_LIFECYCLE_MAPPING_JSON"),
		}
	}},
}

const controlledCommitOutcomeIndeterminateExitCode = 75

func main() {
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Overload(".env")
	}

	dryRun := flag.Bool("dry-run", false, "Print migration status and exit without applying anything")
	rollback := flag.Bool("rollback", false, "Roll back the most recently applied migration instead of applying pending ones")
	printCeiling := flag.Bool("print-ceiling", false, "Print the highest registered migration version and exit")
	printVersions := flag.Bool("print-versions", false, "Print the complete registered migration version list as JSON and exit")
	printReleaseSequence := flag.Bool("print-release-sequence", false, "Print the ordered exact migrations required by this release, one per line, and exit")
	exactVersion := flag.String("exact-version", "", "Register and run exactly one approved production migration")
	flag.Parse()
	if *printReleaseSequence {
		if _, err := validateArtifactMigrationReleaseSequence(artifactMigrationReleaseSequence); err != nil {
			log.Fatal("Invalid artifact migration release sequence:", err)
		}
		for _, releaseVersion := range artifactMigrationReleaseSequence {
			fmt.Fprintln(os.Stdout, releaseVersion)
		}
		return
	}
	if *printVersions {
		m := migration.NewMigrator(nil)
		if err := registerSelectedMigrations(m, *exactVersion); err != nil {
			log.Fatal("Select migrations:", err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(m.RegisteredVersions()); err != nil {
			log.Fatal("Print migration versions:", err)
		}
		return
	}
	if *printCeiling {
		m := migration.NewMigrator(nil)
		if err := registerSelectedMigrations(m, *exactVersion); err != nil {
			log.Fatal("Select migrations:", err)
		}
		fmt.Println(m.Ceiling())
		return
	}
	expectedCeiling := os.Getenv("EXPECTED_MIGRATION_CEILING")
	predecessor, err := validateExactMigrationOptions(*exactVersion, expectedCeiling, *rollback)
	if err != nil {
		log.Fatal("Invalid migration selection:", err)
	}

	dbURL, err := migration.ResolveMigrationDSN(migration.ResolveMigrationDSNInput{
		AppEnv:               os.Getenv("APP_ENV"),
		MigrationDatabaseURL: os.Getenv("MIGRATION_DATABASE_URL"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
	})
	if err != nil {
		log.Fatal("Migration database URL:", err)
	}
	lockTimeout, err := migration.ParseAdvisoryLockTimeout(os.Getenv("MIGRATION_ADVISORY_LOCK_TIMEOUT"))
	if err != nil {
		log.Fatal("Migration lock timeout:", err)
	}

	provenanceURL, err := buildinfo.DatabaseURL(dbURL, version, os.Getenv("INVOCATION_ID"))
	if err != nil {
		log.Fatal("Invalid migration database URL:", err)
	}
	db, err := sql.Open("pgx", provenanceURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	m := migration.NewMigrator(db)
	if err := m.SetAdvisoryLockTimeout(lockTimeout); err != nil {
		log.Fatal("Set advisory lock timeout:", err)
	}
	// Register in version order. Order matters: each migration is
	// recorded in the `migrations` tracking table with its version, and
	// the runner refuses to re-apply an already-recorded version.
	if err := registerSelectedMigrations(m, *exactVersion); err != nil {
		log.Fatal("Select migrations:", err)
	}
	if predecessor != "" {
		if err := requireAppliedMigration(db, predecessor); err != nil {
			log.Fatal("Validate exact migration predecessor:", err)
		}
	}

	switch {
	case *rollback:
		if err := m.Rollback(); err != nil {
			log.Fatal("Rollback failed:", err)
		}
		fmt.Println("Rollback complete.")
	case *dryRun:
		status, err := m.GetStatus()
		if err != nil {
			log.Fatal("Failed to get status:", err)
		}
		printStatus(status)
	default:
		if err := m.MigrateWithExpectedCeiling(expectedCeiling); err != nil {
			log.Print("Migration failed:", err)
			os.Exit(migrationFailureExitCode(err))
		}
		status, err := m.GetStatus()
		if err != nil {
			log.Fatal("Failed to get status after migrate:", err)
		}
		printStatus(status)
	}
}

func validateExactMigrationOptions(exactVersion, expectedCeiling string, rollback bool) (string, error) {
	if exactVersion == "" {
		return "", nil
	}
	if rollback {
		return "", fmt.Errorf("exact-version cannot be combined with rollback")
	}
	if expectedCeiling != exactVersion {
		return "", fmt.Errorf("exact-version %s requires EXPECTED_MIGRATION_CEILING=%s", exactVersion, exactVersion)
	}
	descriptor, ok := migrationDescriptorForVersion(exactVersion)
	if !ok || !descriptor.exactDeploy {
		return "", fmt.Errorf("unsupported exact migration version %q", exactVersion)
	}
	return descriptor.predecessor, nil
}

func requireAppliedMigration(db *sql.DB, version string) error {
	var applied bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM public.migrations WHERE version = $1)`, version).Scan(&applied); err != nil {
		return fmt.Errorf("query migration %s provenance: %w", version, err)
	}
	if !applied {
		return fmt.Errorf("migration %s must be applied before this exact migration", version)
	}
	return nil
}

func migrationFailureExitCode(err error) int {
	if migration.IsControlledCommitOutcomeIndeterminate(err) {
		return controlledCommitOutcomeIndeterminateExitCode
	}
	return 1
}

func artifactMigrationCeiling() string {
	ceiling, err := validateArtifactMigrationReleaseSequence(artifactMigrationReleaseSequence)
	if err != nil {
		panic(err)
	}
	return ceiling
}

func validateArtifactMigrationReleaseSequence(sequence []string) (string, error) {
	if len(sequence) == 0 {
		return "", fmt.Errorf("release sequence is empty")
	}
	for index, version := range sequence {
		descriptor, ok := migrationDescriptorForVersion(version)
		if !ok {
			return "", fmt.Errorf("release sequence contains unknown migration %q", version)
		}
		if !descriptor.exactDeploy {
			return "", fmt.Errorf("release sequence migration %q is not exact-deployable", version)
		}
		if index > 0 && descriptor.predecessor != sequence[index-1] {
			return "", fmt.Errorf(
				"release sequence migration %q requires predecessor %q, not %q",
				version,
				descriptor.predecessor,
				sequence[index-1],
			)
		}
	}
	ceiling := sequence[len(sequence)-1]
	descriptor, _ := migrationDescriptorForVersion(ceiling)
	if !descriptor.artifactCeiling {
		return "", fmt.Errorf("release sequence ceiling %q is not an artifact ceiling", ceiling)
	}
	return ceiling, nil
}

// registerMigrations selects the current artifact ceiling from migrationRegistry.
// The CI guard compares that registry with the Go migration implementations.
func registerMigrations(m *migration.Migrator) {
	if err := registerMigrationsForArtifact(m, artifactMigrationCeiling()); err != nil {
		panic(err)
	}
}

func registerMigrationsForArtifact(m *migration.Migrator, ceiling string) error {
	ceilingDescriptor, ok := migrationDescriptorForVersion(ceiling)
	if !ok || !ceilingDescriptor.artifactCeiling {
		return fmt.Errorf("unsupported compiled migration ceiling %q", ceiling)
	}
	for _, descriptor := range migrationRegistry {
		if descriptor.version > ceiling {
			break
		}
		m.Register(descriptor.newMigrationFunc())
	}
	return nil
}

func registerSelectedMigrations(m *migration.Migrator, exactVersion string) error {
	if exactVersion == "" {
		registerMigrations(m)
		return nil
	}
	descriptor, ok := migrationDescriptorForVersion(exactVersion)
	if !ok || !descriptor.exactDeploy {
		return fmt.Errorf("unsupported exact migration version %q", exactVersion)
	}
	m.Register(descriptor.newMigrationFunc())
	return nil
}

func migrationDescriptorForVersion(version string) (migrationDescriptor, bool) {
	for _, descriptor := range migrationRegistry {
		if descriptor.version == version {
			return descriptor, true
		}
	}
	return migrationDescriptor{}, false
}

func printStatus(status []migration.MigrationStatus) {
	fmt.Println("\nMigration Status:")
	for _, s := range status {
		executed := "not run"
		if s.ExecutedAt != nil {
			executed = s.ExecutedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("  %s  %-8s  %s  (executed: %s)\n",
			s.Version, s.Status, s.Name, executed)
	}
	fmt.Println()
}
