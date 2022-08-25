package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/api/rest/v1/users"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	greetersGRPC "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/greeter"
	profilesGRPC "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/profiles"
	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

// TODO: unify gRPC implementation
type server struct {
	profilesGRPC.UnimplementedProfilesServiceServer
}

func (s *server) ValidateCredentials(ctx context.Context, in *profilesGRPC.ValidateCredentialsRequest) (*profilesGRPC.ValidateCredentialsReply, error) {
	log.Printf("Login: %v", in.GetLogin())       // todo clean
	log.Printf("Password: %v", in.GetPassword()) // todo clean
	result, err := credentials.CheckCredentials(in.GetLogin(), in.GetPassword())
	if err != nil {
		return nil, err
	}

	return &profilesGRPC.ValidateCredentialsReply{UserId: int32(result.UserId), IsValid: result.IsValid}, nil
}

type server2 struct {
	greetersGRPC.UnimplementedGreeterServer
}

func (s *server2) SayHello(ctx context.Context, in *greetersGRPC.HelloRequest) (*greetersGRPC.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &greetersGRPC.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func Start() {
	app.LoadEnv()
	serviceServer := &server{}
	serviceServer2 := &server2{}

	registerServices := func(s *grpc.Server) {
		profilesGRPC.RegisterProfilesServiceServer(s, serviceServer)
		greetersGRPC.RegisterGreeterServer(s, serviceServer2)
	}

	// TODO: add env var with paths to certs
	creds, err := app.LoadTLSCredentialsForServer("configs/tls/server-cert.pem", "configs/tls/server-key.pem")
	if err != nil {
		log.Fatalf("unable to load TLS credentials")
	}

	go func() {
		app.StartGRPC(setup, shutdown, app.HostGRPC(), registerServices, &creds)
	}()
	app.StartHTTP(setup, shutdown, app.HostHTTP(), router())
}

func setup() {
	services.Instance()
}

func shutdown() {
	services.Instance().Shutdown()
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

// TODO: unify
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
		verificationResult, err := services.Instance().AuthGRPC().VerifyToken(token)

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
