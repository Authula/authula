package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-bun/bunschema"
	_ "ariga.io/atlas/sdk/recordriver"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
)

func main() {
	// Parse command line flags
	dialectFlag := flag.String("dialect", "sqlite", "Database dialect (sqlite, postgres, mysql)")
	flag.Parse()

	// Map flag to bunschema dialect
	dialect := bunschema.DialectSQLite
	switch *dialectFlag {
	case "sqlite":
		dialect = bunschema.DialectSQLite
	case "postgres":
		dialect = bunschema.DialectPostgres
	case "mysql":
		dialect = bunschema.DialectMySQL
	}

	stmts, err := bunschema.New(dialect).Load(
		&types.JWKS{},
		&types.RefreshToken{},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load bun schema: %v\n", err)
		os.Exit(1)
	}

	io.WriteString(os.Stdout, stmts)
}
