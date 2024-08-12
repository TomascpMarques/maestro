package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/TomascpMarques/maestro/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var db *sqlx.DB = sqlx.MustOpen("sqlite3", "local_test")
	err := db.Ping()
	if err != nil {
		log.Fatalf("Error getting the database")
	}

	schema := `CREATE TABLE spo (
    value integer,
    time text
    );`

	// execute a query on the server
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("Could not build")
	}

	_, someErr := telemetry.InitTelemetry()
	if errors.Is(someErr, telemetry.ErrFailedToRegister) {
		log.Println("OK")
	}

	app := gin.Default()

	api := app.Group("/api")

	HttpApi(api)

	app.Run() // listen and serve on 0.0.0.0:8080
}

func HttpApi(api *gin.RouterGroup) {
	v1 := api.Group("/v1")

	dataSources := v1.Group("/source")

	pmds := dataSources.Group("/pmd")

	pmds.POST("/spo2", func(ctx *gin.Context) {
		type Spo2Measure struct {
			Value      int    `form:"value"  binding:"required"`
			SourceID   string `form:"source" binding:"required"`
			DeviceType string `form:"device" binding:"required"`
		}

		var spo2Measure Spo2Measure
		if err := ctx.ShouldBind(&spo2Measure); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid data",
			})
			log.Printf("Erro: %s", err.Error())
			return
		}

		ctx.Status(http.StatusAccepted)
	})
}
