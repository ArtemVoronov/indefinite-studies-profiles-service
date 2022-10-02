package services

import (
	"fmt"
	"sync"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	auth "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	notifications "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/notifications"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/subscriptions"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/whitelist"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

type Services struct {
	auth          *auth.AuthGRPCService
	db            *db.PostgreSQLService
	feed          *feed.FeedBuilderGRPCService
	whitelist     *whitelist.WhiteListService
	notifications *notifications.NotificationsGRPCService
	subscriptions *subscriptions.SubscriptionsGRPCService
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
	feedcreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("FEED_SERVICE_CLIENT_TLS_CERT_PATH"))
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

	return &Services{
		auth:          auth.CreateAuthGRPCService(utils.EnvVar("AUTH_SERVICE_GRPC_HOST")+":"+utils.EnvVar("AUTH_SERVICE_GRPC_PORT"), &authcreds),
		feed:          feed.CreateFeedBuilderGRPCService(utils.EnvVar("FEED_SERVICE_GRPC_HOST")+":"+utils.EnvVar("FEED_SERVICE_GRPC_PORT"), &feedcreds),
		db:            db.CreatePostgreSQLService(),
		whitelist:     whitelist.CreateWhiteListService(utils.EnvVar("APP_WHITE_LIST_PATH")),
		notifications: notifications.CreateNotificationsGRPCService(utils.EnvVar("NOTIFICATIONS_SERVICE_GRPC_HOST")+":"+utils.EnvVar("NOTIFICATIONS_SERVICE_GRPC_PORT"), &notificationscreds),
		subscriptions: subscriptions.CreateSubscriptionsGRPCService(utils.EnvVar("SUBSCRIPTIONS_SERVICE_GRPC_HOST")+":"+utils.EnvVar("SUBSCRIPTIONS_SERVICE_GRPC_PORT"), &subscriptionscreds),
	}
}

func (s *Services) Shutdown() error {
	result := []error{}
	err := s.auth.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.db.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.feed.Shutdown()
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
	if len(result) > 0 {
		return fmt.Errorf("errors during shutdown: %v", result)
	}
	return nil
}

func (s *Services) DB() *db.PostgreSQLService {
	return s.db
}

func (s *Services) Auth() *auth.AuthGRPCService {
	return s.auth
}

func (s *Services) Feed() *feed.FeedBuilderGRPCService {
	return s.feed
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
