package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/users"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func Start() {
	setup()
	defer shutdown()
	srv := &http.Server{
		Addr:    host(),
		Handler: router(),
	}

	go func() {
		log.Printf("App starting at localhost%s ...\n", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Println("Server was closed")
		} else if err != nil {
			log.Fatalf("Unable to start app: %v\n", err)
		}
	}()

	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server has been shutdown")
}

func setup() {
	loadEnv()
	db.Instance()
	auth.Instance()
}

func shutdown() {
	db.Instance().Shutdown()
	auth.Instance().Shutdown()
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Print("No .env file found")
	}
}

func host() string {
	port := utils.EnvVarDefault("APP_PORT", "3005")
	host := ":" + port
	return host
}

func mode() string {
	return utils.EnvVarDefault("APP_MODE", "debug")
}

func router() *gin.Engine {
	router := gin.Default()
	gin.SetMode(mode())
	router.Use(cors())
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

func cors() gin.HandlerFunc {
	cors := utils.EnvVarDefault("CORS", "*")
	return func(c *gin.Context) {
		c.Writer.Header().Add("Access-Control-Allow-Origin", cors)
		c.Next()
	}
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
