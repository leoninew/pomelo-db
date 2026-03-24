package db

import (
	"testing"

	"github.com/mingyuan/pomelo-db/internal/config"
)

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.DatasourceConfig
		want    string
		wantErr bool
	}{
		{
			name: "mysql basic",
			cfg: &config.DatasourceConfig{
				Type:     "mysql",
				Host:     "127.0.0.1",
				Port:     3306,
				Database: "testdb",
				User:     "root",
				Password: "secret",
				Options:  map[string]string{},
			},
			want: "root:secret@tcp(127.0.0.1:3306)/testdb?parseTime=true",
		},
		{
			name: "sqlserver",
			cfg: &config.DatasourceConfig{
				Type:     "sqlserver",
				Host:     "127.0.0.1",
				Port:     1433,
				Database: "testdb",
				User:     "sa",
				Password: "secret",
				Options:  map[string]string{},
			},
			want: "sqlserver://sa:secret@127.0.0.1:1433?database=testdb",
		},
		{
			name: "dm",
			cfg: &config.DatasourceConfig{
				Type:     "dm",
				Host:     "127.0.0.1",
				Port:     5236,
				Database: "testdb",
				User:     "SYSDBA",
				Password: "secret",
				Options:  map[string]string{},
			},
			want: "dm://SYSDBA:secret@127.0.0.1:5236/testdb",
		},
		{
			name: "vastbase with schema",
			cfg: &config.DatasourceConfig{
				Type:     "vastbase",
				Host:     "127.0.0.1",
				Port:     5432,
				Database: "testdb",
				User:     "vbadmin",
				Password: "secret",
				Options:  map[string]string{"schema": "public"},
			},
			want: "host=127.0.0.1 port=5432 user=vbadmin password=secret dbname=testdb search_path=public TimeZone=Asia/Shanghai sslmode=disable",
		},
		{
			name: "vastbase without schema",
			cfg: &config.DatasourceConfig{
				Type:     "vastbase",
				Host:     "127.0.0.1",
				Port:     5432,
				Database: "testdb",
				User:     "vbadmin",
				Password: "secret",
				Options:  map[string]string{},
			},
			want: "host=127.0.0.1 port=5432 user=vbadmin password=secret dbname=testdb TimeZone=Asia/Shanghai sslmode=disable",
		},
		{
			name: "opengauss with schema",
			cfg: &config.DatasourceConfig{
				Type:     "opengauss",
				Host:     "127.0.0.1",
				Port:     5432,
				Database: "testdb",
				User:     "gaussdb",
				Password: "secret",
				Options:  map[string]string{"schema": "myschema"},
			},
			want: "host=127.0.0.1 port=5432 user=gaussdb password=secret dbname=testdb search_path=myschema TimeZone=Asia/Shanghai sslmode=disable",
		},
		{
			name: "sqlite",
			cfg: &config.DatasourceConfig{
				Type:     "sqlite",
				Database: "/path/to/test.db",
			},
			want: "/path/to/test.db",
		},
		{
			name: "unsupported type",
			cfg: &config.DatasourceConfig{
				Type:     "postgresql",
				Host:     "127.0.0.1",
				Port:     5432,
				Database: "testdb",
				User:     "postgres",
				Password: "secret",
				Options:  map[string]string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildDSN(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildDSN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("buildDSN() = %v, want %v", got, tt.want)
			}
		})
	}
}
