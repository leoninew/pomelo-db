package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		want    *DatasourceConfig
		wantErr bool
	}{
		{
			name: "mysql basic",
			dsn:  "mysql://root:secret@127.0.0.1:3306/mydb",
			want: &DatasourceConfig{
				Type:     "mysql",
				Host:     "127.0.0.1",
				Port:     3306,
				Database: "mydb",
				User:     "root",
				Password: "secret",
				Options:  map[string]string{},
			},
		},
		{
			name: "sqlserver",
			dsn:  "sqlserver://sa:secret@127.0.0.1:1433/mydb",
			want: &DatasourceConfig{
				Type:     "sqlserver",
				Host:     "127.0.0.1",
				Port:     1433,
				Database: "mydb",
				User:     "sa",
				Password: "secret",
				Options:  map[string]string{},
			},
		},
		{
			name: "dm",
			dsn:  "dm://SYSDBA:secret@127.0.0.1:5236/mydb",
			want: &DatasourceConfig{
				Type:     "dm",
				Host:     "127.0.0.1",
				Port:     5236,
				Database: "mydb",
				User:     "SYSDBA",
				Password: "secret",
				Options:  map[string]string{},
			},
		},
		{
			name: "vastbase with schema",
			dsn:  "vastbase://vbadmin:secret@127.0.0.1:5432/mydb?schema=public",
			want: &DatasourceConfig{
				Type:     "vastbase",
				Host:     "127.0.0.1",
				Port:     5432,
				Database: "mydb",
				User:     "vbadmin",
				Password: "secret",
				Options:  map[string]string{"schema": "public"},
			},
		},
		{
			name: "opengauss without schema",
			dsn:  "opengauss://gaussdb:secret@127.0.0.1:5432/mydb",
			want: &DatasourceConfig{
				Type:     "opengauss",
				Host:     "127.0.0.1",
				Port:     5432,
				Database: "mydb",
				User:     "gaussdb",
				Password: "secret",
				Options:  map[string]string{},
			},
		},
		{
			name: "sqlite windows absolute path",
			dsn:  "sqlite://C:/data/example.db",
			want: &DatasourceConfig{
				Type:     "sqlite",
				Database: "C:/data/example.db",
			},
		},
		{
			name: "sqlite linux absolute path",
			dsn:  "sqlite:///var/data/example.db",
			want: &DatasourceConfig{
				Type:     "sqlite",
				Database: "/var/data/example.db",
			},
		},
		{
			name: "sqlite relative path",
			dsn:  "sqlite://./data/example.db",
			want: &DatasourceConfig{
				Type:     "sqlite",
				Database: "./data/example.db",
			},
		},
		{
			name: "password with percent-encoded special chars",
			dsn:  "mysql://root:p%40ss%3Aw0rd%21@127.0.0.1:3306/mydb",
			want: &DatasourceConfig{
				Type:     "mysql",
				Host:     "127.0.0.1",
				Port:     3306,
				Database: "mydb",
				User:     "root",
				Password: "p@ss:w0rd!",
				Options:  map[string]string{},
			},
		},
		{
			name: "multiple query parameters",
			dsn:  "mysql://root:secret@127.0.0.1:3306/mydb?charset=utf8mb4&timeout=30s&schema=public",
			want: &DatasourceConfig{
				Type:     "mysql",
				Host:     "127.0.0.1",
				Port:     3306,
				Database: "mydb",
				User:     "root",
				Password: "secret",
				Options: map[string]string{
					"charset": "utf8mb4",
					"timeout": "30s",
					"schema":  "public",
				},
			},
		},
		{
			name:    "invalid dsn",
			dsn:     "not-a-valid-dsn",
			wantErr: true,
		},
		{
			name:    "invalid port",
			dsn:     "mysql://root:secret@127.0.0.1:abc/mydb",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDSN(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDSN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Host != tt.want.Host {
				t.Errorf("Host = %v, want %v", got.Host, tt.want.Host)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %v, want %v", got.Port, tt.want.Port)
			}
			if got.Database != tt.want.Database {
				t.Errorf("Database = %v, want %v", got.Database, tt.want.Database)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %v, want %v", got.User, tt.want.User)
			}
			if got.Password != tt.want.Password {
				t.Errorf("Password = %v, want %v", got.Password, tt.want.Password)
			}
			if !reflect.DeepEqual(got.Options, tt.want.Options) {
				t.Errorf("Options = %v, want %v", got.Options, tt.want.Options)
			}
		})
	}
}

var testDefaults = []byte(`# Default configuration - embedded into binary, no user action needed
log:
  level: warn

query:
  readonly: true
  datasources: {}
`)

func TestLoadConfigWithDSN(t *testing.T) {
	// Create a temporary directory and change to it
	tmpDir := t.TempDir()

	// Create .env file with datasource DSNs
	envContent := `POMELO_DB_TEST_MYSQL=mysql://root:secret@127.0.0.1:3306/testdb
POMELO_DB_TEST_SQLITE=sqlite:///tmp/test.db
POMELO_DB_TEST_VASTBASE=vastbase://vbadmin:secret@127.0.0.1:5432/testdb?schema=public
`
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	// Save current directory and change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Load configuration
	cfg, err := Load(testDefaults)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test DSN-based datasource (names are lowercased from POMELO_DB_* prefix)
	t.Run("DSN MySQL", func(t *testing.T) {
		ds, err := cfg.GetDatasource("test_mysql")
		if err != nil {
			t.Fatalf("GetDatasource() error = %v", err)
		}
		if ds.Type != "mysql" {
			t.Errorf("Type = %v, want mysql", ds.Type)
		}
		if ds.Host != "127.0.0.1" {
			t.Errorf("Host = %v, want 127.0.0.1", ds.Host)
		}
		if ds.Port != 3306 {
			t.Errorf("Port = %v, want 3306", ds.Port)
		}
		if ds.Database != "testdb" {
			t.Errorf("Database = %v, want testdb", ds.Database)
		}
		if ds.User != "root" {
			t.Errorf("User = %v, want root", ds.User)
		}
		if ds.Password != "secret" {
			t.Errorf("Password = %v, want secret", ds.Password)
		}
	})

	t.Run("DSN SQLite", func(t *testing.T) {
		ds, err := cfg.GetDatasource("test_sqlite")
		if err != nil {
			t.Fatalf("GetDatasource() error = %v", err)
		}
		if ds.Type != "sqlite" {
			t.Errorf("Type = %v, want sqlite", ds.Type)
		}
		// SQLite paths are converted to absolute on Windows, just check type
	})

	t.Run("DSN Vastbase with schema", func(t *testing.T) {
		ds, err := cfg.GetDatasource("test_vastbase")
		if err != nil {
			t.Fatalf("GetDatasource() error = %v", err)
		}
		if ds.Type != "vastbase" {
			t.Errorf("Type = %v, want vastbase", ds.Type)
		}
		if schema := ds.Options["schema"]; schema != "public" {
			t.Errorf("Options[schema] = %v, want public", schema)
		}
	})

	t.Run("Non-existent datasource", func(t *testing.T) {
		_, err := cfg.GetDatasource("test_nonexistent")
		if err == nil {
			t.Error("expected error for non-existent datasource")
		}
	})
}
