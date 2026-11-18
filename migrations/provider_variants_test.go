package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForProvider(t *testing.T) {
	t.Parallel()

	postgresMigrations := []Migration{{Version: "001_postgres"}}
	mysqlMigrations := []Migration{{Version: "001_mysql"}}

	tests := []struct {
		name      string
		provider  string
		variants  func(calls *int) ProviderVariants
		want      []Migration
		wantCalls int
	}{
		{
			name:     "empty variants returns nil",
			provider: "postgres",
			variants: func(*int) ProviderVariants {
				return nil
			},
			want: nil,
		},
		{
			name:     "empty provider returns nil",
			provider: "",
			variants: func(calls *int) ProviderVariants {
				return ProviderVariants{
					"postgres": func() []Migration {
						*calls++
						return postgresMigrations
					},
				}
			},
			want: nil,
		},
		{
			name:     "whitespace provider returns nil",
			provider: "   \t ",
			variants: func(calls *int) ProviderVariants {
				return ProviderVariants{
					"postgres": func() []Migration {
						*calls++
						return postgresMigrations
					},
				}
			},
			want: nil,
		},
		{
			name:     "exact provider match returns constructor result",
			provider: "postgres",
			variants: func(calls *int) ProviderVariants {
				return ProviderVariants{
					"postgres": func() []Migration {
						*calls++
						return postgresMigrations
					},
					"mysql": func() []Migration {
						return mysqlMigrations
					},
				}
			},
			want:      postgresMigrations,
			wantCalls: 1,
		},
		{
			name:     "provider is trimmed and lowercased",
			provider: "  MySQL  ",
			variants: func(calls *int) ProviderVariants {
				return ProviderVariants{
					"mysql": func() []Migration {
						*calls++
						return mysqlMigrations
					},
				}
			},
			want:      mysqlMigrations,
			wantCalls: 1,
		},
		{
			name:     "unknown provider returns nil",
			provider: "oracle",
			variants: func(calls *int) ProviderVariants {
				return ProviderVariants{
					"postgres": func() []Migration {
						*calls++
						return postgresMigrations
					},
				}
			},
			want: nil,
		},
		{
			name:     "nil constructor returns nil",
			provider: "mysql",
			variants: func(*int) ProviderVariants {
				return ProviderVariants{
					"mysql": nil,
				}
			},
			want: nil,
		},
		{
			name:     "empty constructor result returns empty slice",
			provider: "postgres",
			variants: func(calls *int) ProviderVariants {
				return ProviderVariants{
					"postgres": func() []Migration {
						*calls++
						return []Migration{}
					},
				}
			},
			want:      []Migration{},
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			got := ForProvider(tt.provider, tt.variants(&calls))

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantCalls, calls)

			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)
		})
	}
}
