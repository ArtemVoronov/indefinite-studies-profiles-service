package app

import (
	"fmt"
	"net/http"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/grpc/v1/profiles"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/users"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"

	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func Start() {
	app.LoadEnv()
	file := log.SetUpLogPath(utils.EnvVarDefault("APP_LOGS_PATH", "stdout"))
	if file != nil {
		defer file.Close()
	}
	creds := app.TLSCredentials()
	go func() {
		app.StartGRPC(setup, shutdown, app.HostGRPC(), createGrpcApi, &creds, log.Instance())
	}()
	app.StartHTTP(setup, shutdown, app.HostHTTP(), createRestApi(log.Instance()))
}

func setup() {
	services.Instance()
}

func shutdown() {
	err := services.Instance().Shutdown()
	log.Error("error during app shutdown", err.Error())
}

func createRestApi(logger *logrus.Logger) *gin.Engine {
	router := gin.Default()
	gin.SetMode(app.Mode())
	router.Use(app.Cors())
	router.Use(app.NewLoggerMiddleware(logger))
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	v1 := router.Group("/api/v1")

	v1.GET("/users/ping", ping.Ping)
	v1.GET("/users/name/:uuid", users.GetUserName)

	v1.POST("/users/signup", users.SignUpStart)
	v1.PUT("/users/signup", users.SignUpFinish)
	v1.POST("/users/signup/resend", users.ResendConfirmationLink)

	v1.POST("/users/password/restore", users.RestorePasswordStart)
	v1.PUT("/users/password/restore", users.RestorePasswordFinish)

	authorized := router.Group("/api/v1")
	authorized.Use(app.AuthReqired(authenicate))
	{
		authorized.GET("/users/debug/vars", app.RequiredOwnerRole(), expvar.Handler())
		authorized.GET("/users/safe-ping", app.RequiredOwnerRole(), ping.SafePing)

		// TODO: add explicit route for changing email with confirmation

		authorized.GET("/users", app.RequiredOwnerRole(), users.GetUsers)
		authorized.GET("/users/:uuid", app.RequiredOwnerRole(), users.GetUser)
		authorized.GET("/users/me", users.GetMyProfile)
		authorized.POST("/users", app.RequiredOwnerRole(), users.CreateUser)
		authorized.PUT("/users", users.UpdateUser)
		authorized.DELETE("/users", users.DeleteUser)
	}

	return router
}

func createGrpcApi(s *grpc.Server) {
	profiles.RegisterServiceServer(s)
}

func authenicate(token string) (*auth.VerificationResult, error) {
	return services.Instance().Auth().VerifyToken(token)
}
