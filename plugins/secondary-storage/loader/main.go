package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-bun/bunschema"
	_ "ariga.io/atlas/sdk/recordriver"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/secondary-storage/types"
)

func main() {
	dialectFlag := flag.String("dialect", "sqlite", "Database dialect (sqlite, postgres, mysql)")
	flag.Parse()

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
		&types.KeyValueStore{},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load bun schema: %v\n", err)
		os.Exit(1)
	}

	io.WriteString(os.Stdout, stmts)
}
