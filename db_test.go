package main

import (
	"atlas-db-poc/pgdb"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	db1 *sql.DB
)

func setupDb(port string) (*sql.DB, func() error) {
	var dbconn *sql.DB
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=user_name",
			"POSTGRES_DB=dbname",
			"listen_addresses = '*'",
		},
		// ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port(port): {
				{HostIP: "0.0.0.0", HostPort: "5432"},
			},
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort(fmt.Sprintf("%s/tcp", port))
	databaseUrl := fmt.Sprintf("postgres://user_name:secret@%s/dbname?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", databaseUrl)

	resource.Expire(120) // Tell docker to hard kill the container in 120 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		dbconn, err = sql.Open("pgx", databaseUrl)
		if err != nil {
			return err
		}
		return dbconn.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	return dbconn, func() error {
		return pool.Purge(resource)
	}
}

func TestMain(m *testing.M) {
	// create a test connection for happy paths
	conn1, close1 := setupDb("5432")
	db1 = conn1

	// migrate schema
	driver, err := postgres.WithInstance(conn1, &postgres.Config{})
	if err != nil {
		log.Fatalf("Could not migrate db driver: %s", err)
	}
	mg, err := migrate.NewWithDatabaseInstance(
		"file://helm/migrations",
		"postgres", driver)
	if err != nil {
		log.Fatalf("Could not create new migrate instance: %s", err)
	}
	if err := mg.Up(); err != nil {
		log.Fatalf("Could not migrate schema: %s", err)
	}

	//Run tests
	code := m.Run()

	// teardown
	mg.Down()

	// You can't defer this because os.Exit doesn't care for defer
	if err := close1(); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestCreateUser(t *testing.T) {
	q := pgdb.New(db1)

	_, err := q.CreateUser(context.Background(), pgdb.CreateUserParams{
		Name: "aaron",
	})

	if err != nil {
		t.Error(err)
	}
}
