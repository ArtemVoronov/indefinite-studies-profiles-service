package app

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/users"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
)

func Start() {
	app.Start(setup, shutdown, app.Host(), router())
}

func setup() {
	db.Instance()
	auth.Instance()
}

func shutdown() {
	db.Instance().Shutdown()
	auth.Instance().Shutdown()
}

func router() *gin.Engine {
	router := gin.Default()
	gin.SetMode(app.Mode())
	router.Use(app.Cors())
	router.Use(gin.Logger())

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// TODO: add permission controller by user role and user state
	v1 := router.Group("/api/v1")

	v1.GET("/ping", ping.Ping)
	authorized := router.Group("/api/v1")
	authorized.Use(authReqired())
	{
		authorized.GET("/debug/vars", expvar.Handler())
		authorized.GET("/safe-ping", ping.SafePing)

		// TODO: add signup

		authorized.GET("/users", users.GetUsers)
		authorized.GET("/users/:id", users.GetUser)
		authorized.POST("/users", users.CreateUser)
		// authorized.PUT("/users/:id", users.UpdateUser) // TODO: make clear updte per fields (optional fields + checking a permission to update the user)
		// authorized.DELETE("/users/:id", users.DeleteUser) // TODO (checking a permission to delete the user)
		authorized.PUT("/users/credentials:validate", users.IsValidCredentials)
		// authorized.PUT("/users/credentials", users.UpdateCredentials) // TODO (checking a permission to update the user)
	}

	return router
}

func authReqired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		// fmt.Println("---------------AuthReqired---------------")
		// fmt.Printf("header: %v\n", header)
		// fmt.Println("---------------AuthReqired---------------")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer") {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		token := authHeader[len("Bearer "):]
		verificationResult, err := auth.Instance().VerifyToken(token)

		if err != nil {
			c.JSON(http.StatusInternalServerError, "Internal Server Error")
			log.Printf("error during verifying access token: %v\n", err)
			c.Abort()
			return
		}

		if (*verificationResult).IsExpired {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		c.Next()
	}
}
