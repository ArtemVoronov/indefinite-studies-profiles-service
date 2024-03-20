package services

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/profiles"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/templates"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	auth "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	notifications "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/notifications"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/shard"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/subscriptions"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/whitelist"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

type Services struct {
	auth          *auth.AuthGRPCService
	whitelist     *whitelist.WhiteListService
	notifications *notifications.NotificationsGRPCService
	subscriptions *subscriptions.SubscriptionsGRPCService
	templates     *templates.EmailTemplateService
	profiles      *profiles.ProfilesService
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
	authcreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("AUTH_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials: %s", err)
	}
	notificationscreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("NOTIFICATIONS_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials: %s", err)
	}
	subscriptionscreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("SUBSCRIPTIONS_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials: %s", err)
	}

	dbClients := []*db.PostgreSQLService{}
	for i := 1; i <= shard.DEFAULT_BUCKET_FACTOR; i++ {
		dbConfig := &db.DBParams{
			Host:         utils.EnvVar("DATABASE_HOST"),
			Port:         utils.EnvVar("DATABASE_PORT"),
			Username:     utils.EnvVar("DATABASE_USER"),
			Password:     utils.EnvVar("DATABASE_PASSWORD"),
			DatabaseName: utils.EnvVar("DATABASE_NAME_PREFIX") + "_" + strconv.Itoa(i),
			SslMode:      utils.EnvVar("DATABASE_SSL_MODE"),
		}
		dbClients = append(dbClients, db.CreatePostgreSQLService(dbConfig))
	}

	return &Services{
		auth:          auth.CreateAuthGRPCService(utils.EnvVar("AUTH_SERVICE_GRPC_HOST")+":"+utils.EnvVar("AUTH_SERVICE_GRPC_PORT"), &authcreds),
		whitelist:     whitelist.CreateWhiteListService(utils.EnvVar("APP_WHITE_LIST_PATH")),
		notifications: notifications.CreateNotificationsGRPCService(utils.EnvVar("NOTIFICATIONS_SERVICE_GRPC_HOST")+":"+utils.EnvVar("NOTIFICATIONS_SERVICE_GRPC_PORT"), &notificationscreds),
		subscriptions: subscriptions.CreateSubscriptionsGRPCService(utils.EnvVar("SUBSCRIPTIONS_SERVICE_GRPC_HOST")+":"+utils.EnvVar("SUBSCRIPTIONS_SERVICE_GRPC_PORT"), &subscriptionscreds),
		templates:     templates.NewEmailTemplateService(utils.EnvVar("TEMPLATES_SERVICE_BASE_URL"), utils.EnvVar("TEMPLATES_SERVICE_SENDER_EMAIL")),
		profiles:      profiles.CreateProfilesService(dbClients),
	}
}

func (s *Services) Shutdown() error {
	result := []error{}
	err := s.auth.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.profiles.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.whitelist.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.notifications.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.subscriptions.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.templates.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	if len(result) > 0 {
		return fmt.Errorf("errors during shutdown: %v", result)
	}
	return nil
}

func (s *Services) Auth() *auth.AuthGRPCService {
	return s.auth
}

func (s *Services) Whitelist() *whitelist.WhiteListService {
	return s.whitelist
}

func (s *Services) Notifications() *notifications.NotificationsGRPCService {
	return s.notifications
}

func (s *Services) Subscriptions() *subscriptions.SubscriptionsGRPCService {
	return s.subscriptions
}

func (s *Services) Templates() *templates.EmailTemplateService {
	return s.templates
}

func (s *Services) Profiles() *profiles.ProfilesService {
	return s.profiles
}
