variable "plugin" {
  type = string
  default = "." # Root app for core migrations
}

# Core migrations use internal/migrations
# Plugin migrations use plugins/{plugin}/migrations

locals {
  migration_dir = var.plugin == "." ? "internal/migrations" : "${var.plugin}/migrations"
}

data "external_schema" "bun_sqlite" {
  program = ["go", "run", "./${var.plugin}/loader", "-dialect", "sqlite"]
}

env "sqlite" {
  src = data.external_schema.bun_sqlite.url
  dev = "sqlite://file?mode=memory"
  migration {
    dir    = "file://${local.migration_dir}/sqlite"
    format = golang-migrate
  }
}

data "external_schema" "bun_postgres" {
  program = ["go", "run", "./${var.plugin}/loader", "-dialect", "postgres"]
}

env "postgres" {
  src = data.external_schema.bun_postgres.url
  dev = "docker://postgres/18/dev"
  migration {
    dir    = "file://${local.migration_dir}/postgres"
    format = golang-migrate
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

data "external_schema" "bun_mysql" {
  program = ["go", "run", "./${var.plugin}/loader", "-dialect", "mysql"]
}

env "mysql" {
  src = data.external_schema.bun_mysql.url
  dev = "docker://mysql/8/dev"
  migration {
    dir    = "file://${local.migration_dir}/mysql"
    format = golang-migrate
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
