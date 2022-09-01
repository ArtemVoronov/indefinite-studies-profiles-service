package app

import (
	"fmt"
	"net/http"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/grpc/v1/profiles"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/users"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func Start() {
	app.LoadEnv()
	creds := app.TLSCredentials()
	go func() {
		app.StartGRPC(setup, shutdown, app.HostGRPC(), createGrpcApi, &creds)
	}()
	app.StartHTTP(setup, shutdown, app.HostHTTP(), createRestApi())
}

func setup() {
	services.Instance()
}

func shutdown() {
	services.Instance().Shutdown()
}

func createRestApi() *gin.Engine {
	router := gin.Default()
	gin.SetMode(app.Mode())
	router.Use(app.Cors())
	router.Use(gin.Logger())
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// TODO: add permission controller by user role and user state
	v1 := router.Group("/api/v1")

	v1.GET("/users/ping", ping.Ping)

	authorized := router.Group("/api/v1")
	authorized.Use(app.AuthReqired(authenicate))
	{
		authorized.GET("/users/debug/vars", expvar.Handler())
		authorized.GET("/users/safe-ping", ping.SafePing)

		// TODO: add explicit route for signup
		// TODO: add explicit route for changing password
		// TODO: add explicit route for changing email with confirmation
		// TODO: add explicit route for restoring password

		authorized.GET("/users", users.GetUsers)
		authorized.GET("/users/:id", users.GetUser)
		authorized.GET("/users/me", users.GetMyProfile)
		authorized.POST("/users", users.CreateUser)   // TODO: check a permission to create the user
		authorized.PUT("/users", users.UpdateUser)    // TODO: check a permission to update the user
		authorized.DELETE("/users", users.DeleteUser) // TODO: check a permission to delete the user
	}

	return router
}

func createGrpcApi(s *grpc.Server) {
	profiles.RegisterServiceServer(s)
}

func authenicate(token string) (*auth.VerificationResult, error) {
	return services.Instance().Auth().VerifyToken(token)
}
