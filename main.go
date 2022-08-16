package main

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/expvar"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/users"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/app"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
)

func main() {
	app.InitEnv()
	host := app.GetHost()
	router := gin.Default()
	router.Use(app.Cors(utils.EnvVar("CORS")))
	router.Use(gin.Logger())

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	db.GetInstance()

	// TODO: add permission controller by user role and user state
	v1 := router.Group("/api/v1")

	v1.GET("/ping", ping.Ping)
	authorized := router.Group("/api/v1")
	authorized.Use(app.AuthReqired())
	{
		authorized.GET("/debug/vars", expvar.Handler())
		authorized.GET("/safe-ping", ping.SafePing)

		// TODO: add signup

		authorized.GET("/users", users.GetUsers)
		authorized.GET("/users/:id", users.GetUser)
		authorized.POST("/users", users.CreateUser)
		authorized.PUT("/users/:id", users.UpdateUser)
		authorized.DELETE("/users/:id", users.DeleteUser)
		authorized.PUT("/users/credentials:validate", users.IsValidCredentials)
		authorized.PUT("/users/credentials", users.UpdateCredentials)
	}

	app.StartServer(host, router)
}
