package postgres

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	migrationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/migrations"
	"github.com/pressly/goose/v3"
)

func TestMigrationStateIsCurrent(t *testing.T) {
	tests := []struct {
		name  string
		state MigrationState
		want  bool
	}{
		{name: "current", state: MigrationState{CurrentVersion: 1, TargetVersion: 1}, want: true},
		{name: "pending", state: MigrationState{CurrentVersion: 1, TargetVersion: 2, HasPending: true}},
		{name: "version mismatch without pending flag", state: MigrationState{CurrentVersion: 1, TargetVersion: 2}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.state.IsCurrent(); got != test.want {
				t.Fatalf("IsCurrent() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestUserRoleMigrationBackfillsExistingUsersAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TALOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	database, err := Open(ctx, testConfig(databaseURL))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	for _, statement := range []string{
		"DROP TABLE IF EXISTS energy_measurements",
		"DROP TABLE IF EXISTS print_job_material_usage",
		"DROP TABLE IF EXISTS job_events",
		"DROP TABLE IF EXISTS print_jobs",
		"DROP TABLE IF EXISTS printers",
		"DROP TABLE IF EXISTS catalog_bom_items",
		"DROP TABLE IF EXISTS supply_movements",
		"DROP TABLE IF EXISTS supplies",
		"DROP TABLE IF EXISTS spool_measurements",
		"DROP TABLE IF EXISTS material_spools",
		"DROP TABLE IF EXISTS materials",
		"DROP TABLE IF EXISTS design_version_files",
		"DROP TABLE IF EXISTS design_versions",
		"DROP TABLE IF EXISTS catalog_parts",
		"DROP TABLE IF EXISTS catalog_items",
		"DROP TABLE IF EXISTS workshop_settings",
		"DROP TABLE IF EXISTS files",
		"DROP TABLE IF EXISTS sessions",
		"DROP TABLE IF EXISTS bootstrap_state",
		"DROP TABLE IF EXISTS client_devices",
		"DROP TABLE IF EXISTS users",
		"DROP TABLE IF EXISTS goose_db_version",
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("reset role migration schema with %q: %v", statement, err)
		}
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrationfiles.Files)
	if err != nil {
		t.Fatalf("create migration provider: %v", err)
	}
	if _, err := provider.UpTo(ctx, 5); err != nil {
		t.Fatalf("migrate through sessions: %v", err)
	}

	var ownerID string
	if err := database.QueryRowContext(
		ctx,
		`INSERT INTO users (name, email_or_username, password_hash, status)
		 VALUES ('Owner', 'owner', '$argon2id$test', 'active') RETURNING id`,
	).Scan(&ownerID); err != nil {
		t.Fatalf("insert pre-role owner: %v", err)
	}
	var viewerID string
	if err := database.QueryRowContext(
		ctx,
		`INSERT INTO users (name, email_or_username, password_hash, status)
		 VALUES ('Existing User', 'existing', '$argon2id$test', 'active') RETURNING id`,
	).Scan(&viewerID); err != nil {
		t.Fatalf("insert pre-role user: %v", err)
	}
	if _, err := database.ExecContext(
		ctx,
		"INSERT INTO bootstrap_state (initial_owner_user_id) VALUES ($1)",
		ownerID,
	); err != nil {
		t.Fatalf("record pre-role bootstrap owner: %v", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("apply role migration: %v", err)
	}

	roles := map[string]string{}
	rows, err := database.QueryContext(ctx, "SELECT id, role FROM users")
	if err != nil {
		t.Fatalf("read migrated roles: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, role string
		if err := rows.Scan(&id, &role); err != nil {
			t.Fatalf("scan migrated role: %v", err)
		}
		roles[id] = role
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migrated roles: %v", err)
	}
	if roles[ownerID] != "owner" || roles[viewerID] != "viewer" {
		t.Fatalf("migrated roles = %#v, want owner/viewer", roles)
	}
}

func TestMigrationLifecycleAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TALOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	database, err := Open(ctx, testConfig(databaseURL))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS energy_measurements"); err != nil {
		t.Fatalf("reset energy measurements schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS print_job_material_usage"); err != nil {
		t.Fatalf("reset job material usage schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS job_events"); err != nil {
		t.Fatalf("reset job events schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS print_jobs"); err != nil {
		t.Fatalf("reset print jobs schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS printers"); err != nil {
		t.Fatalf("reset printers schema: %v", err)
	}

	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS catalog_bom_items"); err != nil {
		t.Fatalf("reset catalog BOM schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS supply_movements"); err != nil {
		t.Fatalf("reset supply movements schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS supplies"); err != nil {
		t.Fatalf("reset supplies schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS spool_measurements"); err != nil {
		t.Fatalf("reset spool measurements schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS material_spools"); err != nil {
		t.Fatalf("reset material spools schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS materials"); err != nil {
		t.Fatalf("reset materials schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS design_version_files"); err != nil {
		t.Fatalf("reset design version files schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS design_versions"); err != nil {
		t.Fatalf("reset design versions schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS catalog_parts"); err != nil {
		t.Fatalf("reset catalog parts schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS catalog_items"); err != nil {
		t.Fatalf("reset catalog items schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS workshop_settings"); err != nil {
		t.Fatalf("reset workshop settings schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS files"); err != nil {
		t.Fatalf("reset files schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS bootstrap_state"); err != nil {
		t.Fatalf("reset bootstrap state schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS sessions"); err != nil {
		t.Fatalf("reset sessions schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS client_devices"); err != nil {
		t.Fatalf("reset client devices schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS users"); err != nil {
		t.Fatalf("reset users schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS goose_db_version"); err != nil {
		t.Fatalf("reset migration state: %v", err)
	}

	state, err := GetMigrationState(ctx, database)
	if err != nil {
		t.Fatalf("GetMigrationState() before migrate error = %v", err)
	}
	if state.CurrentVersion != 0 || state.TargetVersion != 17 || !state.HasPending {
		t.Fatalf("state before migrate = %+v, want current 0, target 17, pending", state)
	}

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	assertMigrationState(t, ctx, database, 17, 17, false)

	results := make(chan error, 2)
	for range 2 {
		go func() {
			results <- Migrate(ctx, database)
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("concurrent Migrate() error = %v", err)
		}
	}

	bootstrap, err := fs.ReadFile(migrationfiles.Files, "00001_bootstrap.sql")
	if err != nil {
		t.Fatalf("read bootstrap migration: %v", err)
	}
	users, err := fs.ReadFile(migrationfiles.Files, "00002_users.sql")
	if err != nil {
		t.Fatalf("read users migration: %v", err)
	}
	bootstrapState, err := fs.ReadFile(migrationfiles.Files, "00003_bootstrap_state.sql")
	if err != nil {
		t.Fatalf("read bootstrap state migration: %v", err)
	}
	clientDevices, err := fs.ReadFile(migrationfiles.Files, "00004_client_devices.sql")
	if err != nil {
		t.Fatalf("read client devices migration: %v", err)
	}
	sessions, err := fs.ReadFile(migrationfiles.Files, "00005_sessions.sql")
	if err != nil {
		t.Fatalf("read sessions migration: %v", err)
	}
	userRoles, err := fs.ReadFile(migrationfiles.Files, "00006_user_roles.sql")
	if err != nil {
		t.Fatalf("read user roles migration: %v", err)
	}
	workshopSettings, err := fs.ReadFile(migrationfiles.Files, "00007_workshop_settings.sql")
	if err != nil {
		t.Fatalf("read workshop settings migration: %v", err)
	}
	files, err := fs.ReadFile(migrationfiles.Files, "00008_files.sql")
	if err != nil {
		t.Fatalf("read files migration: %v", err)
	}
	catalogItems, err := fs.ReadFile(migrationfiles.Files, "00009_catalog_items.sql")
	if err != nil {
		t.Fatalf("read catalog items migration: %v", err)
	}
	catalogDesigns, err := fs.ReadFile(migrationfiles.Files, "00010_catalog_designs.sql")
	if err != nil {
		t.Fatalf("read catalog designs migration: %v", err)
	}
	filamentInventory, err := fs.ReadFile(migrationfiles.Files, "00011_filament_inventory.sql")
	if err != nil {
		t.Fatalf("read filament inventory migration: %v", err)
	}
	supplyInventory, err := fs.ReadFile(migrationfiles.Files, "00012_supply_inventory.sql")
	if err != nil {
		t.Fatalf("read supply inventory migration: %v", err)
	}
	catalogBOM, err := fs.ReadFile(migrationfiles.Files, "00013_catalog_bom.sql")
	if err != nil {
		t.Fatalf("read catalog BOM migration: %v", err)
	}
	printers, err := fs.ReadFile(migrationfiles.Files, "00014_printers.sql")
	if err != nil {
		t.Fatalf("read printers migration: %v", err)
	}
	printJobs, err := fs.ReadFile(migrationfiles.Files, "00015_print_jobs.sql")
	if err != nil {
		t.Fatalf("read print jobs migration: %v", err)
	}
	jobMaterialUsage, err := fs.ReadFile(migrationfiles.Files, "00016_job_material_usage.sql")
	if err != nil {
		t.Fatalf("read job material usage migration: %v", err)
	}
	energyMeasurements, err := fs.ReadFile(migrationfiles.Files, "00017_energy_measurements.sql")
	if err != nil {
		t.Fatalf("read energy measurements migration: %v", err)
	}
	failingMigrations := fstest.MapFS{
		"00001_bootstrap.sql":           {Data: bootstrap},
		"00002_users.sql":               {Data: users},
		"00003_bootstrap_state.sql":     {Data: bootstrapState},
		"00004_client_devices.sql":      {Data: clientDevices},
		"00005_sessions.sql":            {Data: sessions},
		"00006_user_roles.sql":          {Data: userRoles},
		"00007_workshop_settings.sql":   {Data: workshopSettings},
		"00008_files.sql":               {Data: files},
		"00009_catalog_items.sql":       {Data: catalogItems},
		"00010_catalog_designs.sql":     {Data: catalogDesigns},
		"00011_filament_inventory.sql":  {Data: filamentInventory},
		"00012_supply_inventory.sql":    {Data: supplyInventory},
		"00013_catalog_bom.sql":         {Data: catalogBOM},
		"00014_printers.sql":            {Data: printers},
		"00015_print_jobs.sql":          {Data: printJobs},
		"00016_job_material_usage.sql":  {Data: jobMaterialUsage},
		"00017_energy_measurements.sql": {Data: energyMeasurements},
		"00018_failure.sql": {
			Data: []byte("-- +goose Up\nSELECT * FROM table_that_does_not_exist;\n"),
		},
	}
	if err := migrate(ctx, database, failingMigrations); err == nil {
		t.Fatal("migrate() error = nil, want failing migration error")
	}
	assertMigrationState(t, ctx, database, 17, 17, false)
}

func assertMigrationState(t *testing.T, ctx context.Context, database *sql.DB, current, target int64, pending bool) {
	t.Helper()

	state, err := GetMigrationState(ctx, database)
	if err != nil {
		t.Fatalf("GetMigrationState() error = %v", err)
	}
	if state.CurrentVersion != current || state.TargetVersion != target || state.HasPending != pending {
		t.Fatalf(
			"migration state = %+v, want current %d, target %d, pending %t",
			state,
			current,
			target,
			pending,
		)
	}
}
