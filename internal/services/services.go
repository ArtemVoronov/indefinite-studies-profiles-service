package services

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	auth "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

type Services struct {
	authREST *auth.AuthService
	authGRPC *auth.AuthGRPCService
	db       *db.PostgreSQLService
}

var once sync.Once
var instance *Services

func Instance() *Services {
	once.Do(func() {
		if instance == nil {
			instance = createServices()
		}
	})
	return instance
}

func createServices() *Services {
	client := &http.Client{
		Timeout: utils.EnvVarDurationDefault("HTTP_CLIENT_REQUEST_TIMEOUT_IN_SECONDS", time.Second, 30*time.Second),
	}
	// TODO: add env var with paths to certs
	creds, err := app.LoadTLSCredentialsForClient("configs/tls/ca-cert.pem")
	if err != nil {
		log.Fatalf("unable to load TLS credentials")
	}

	return &Services{
		authREST: auth.CreateAuthService(client, utils.EnvVar("AUTH_SERVICE_BASE_URL")),
		authGRPC: auth.CreateAuthGRPCService(utils.EnvVar("AUTH_SERVICE_GRPC_HOST")+":"+utils.EnvVar("AUTH_SERVICE_GRPC_PORT"), &creds),
		db:       db.CreatePostgreSQLService(),
	}
}

func (s *Services) Shutdown() {
	s.authREST.Shutdown()
	s.db.Shutdown()
}

func (s *Services) DB() *db.PostgreSQLService {
	return s.db
}

func (s *Services) AuthREST() *auth.AuthService {
	return s.authREST
}

func (s *Services) AuthGRPC() *auth.AuthGRPCService {
	return s.authGRPC
}
