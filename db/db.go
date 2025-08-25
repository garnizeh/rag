package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

//go:embed seed/*.*
var SeedFiles embed.FS
