package main

import (
	"atlas-db-poc/pgdb"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_DB"),
	)
	db, err := sql.Open("pgx", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	q := pgdb.New(db)

	r := gin.Default()
	r.POST("/create", func(c *gin.Context) {
		var json struct {
			Name string `json:"name" binding:"required"`
		}

		if err := c.Bind(&json); err != nil {
			c.AbortWithStatusJSON(400, err)
			return
		}

		u, err := q.CreateUser(c, pgdb.CreateUserParams{ID: json.Name, Name: json.Name})
		if err != nil {
			c.AbortWithStatusJSON(400, err)
			return
		}
		c.JSON(http.StatusOK, u)

	})
	r.Run(":9000")
}
