package migrations

import (
	"context"
	"fmt"
	"net/url"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/mysql"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func RunCoreMigrations(
	ctx context.Context,
	logger models.Logger,
	provider string,
	dbURL string,
) error {
	u, _ := url.Parse(dbURL)
	db := dbmate.New(u)
	sqlFs, err := GetMigrations(ctx, provider)
	if err != nil {
		return err
	}
	db.FS = *sqlFs
	db.MigrationsDir = []string{"migrations/" + provider}

	return RunMigrations(ctx, logger, db)
}

func DropCoreMigrations(
	ctx context.Context,
	logger models.Logger,
	provider string,
	dbURL string,
) error {
	u, _ := url.Parse(dbURL)
	db := dbmate.New(u)
	sqlFs, err := GetMigrations(ctx, provider)
	if err != nil {
		return err
	}
	db.FS = *sqlFs
	db.MigrationsDir = []string{"migrations/" + provider}
	return DropMigrations(ctx, logger, db)
}

func RunMigrations(
	ctx context.Context,
	logger models.Logger,
	db *dbmate.DB,
) error {
	migrations, err := db.FindMigrations()
	if err != nil {
		panic(err)
	}
	for _, m := range migrations {
		fmt.Println(m.Version, m.FilePath)
	}

	if err := db.CreateAndMigrate(); err != nil {
		return err
	}

	return nil
}

func DropMigrations(ctx context.Context,
	logger models.Logger,
	db *dbmate.DB,
) error {
	migrations, err := db.FindMigrations()
	if err != nil {
		panic(err)
	}
	for _, m := range migrations {
		fmt.Println(m.Version, m.FilePath)
	}

	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		fmt.Printf("Rolling back: %s\n", m.FilePath)
		if err := db.Rollback(); err != nil {
			return err
		}
		fmt.Printf("Rolled back: %s\n", m.FilePath)
	}

	return nil
}
