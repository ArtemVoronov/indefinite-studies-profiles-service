package profiles

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/profiles"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: unify all CRUD ops of users into ProfilesService

type ProfilesServiceServer struct {
	profiles.UnimplementedProfilesServiceServer
}

func RegisterServiceServer(s *grpc.Server) {
	profiles.RegisterProfilesServiceServer(s, &ProfilesServiceServer{})
}

func (s *ProfilesServiceServer) ValidateCredentials(ctx context.Context, in *profiles.ValidateCredentialsRequest) (*profiles.ValidateCredentialsReply, error) {
	result, err := credentials.CheckCredentials(in.GetLogin(), in.GetPassword())
	if err != nil {
		return nil, err
	}

	return &profiles.ValidateCredentialsReply{UserId: int32(result.UserId), IsValid: result.IsValid, Role: result.Role}, nil
}

func (s *ProfilesServiceServer) GetUser(ctx context.Context, in *profiles.GetUserRequest) (*profiles.GetUserReply, error) {
	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		user, err := queries.GetUser(tx, ctx, int(in.GetId()))
		return user, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			return &profiles.GetUserReply{}, nil
		} else {
			return &profiles.GetUserReply{}, err
		}
	}

	user, ok := data.(entities.User)
	if !ok {
		log.Error("Unable to get to user", api.ERROR_ASSERT_RESULT_TYPE)
		return &profiles.GetUserReply{}, err
	}

	return toGetUserReply(&user), nil
}

func (s *ProfilesServiceServer) GetUsers(ctx context.Context, in *profiles.GetUsersRequest) (*profiles.GetUsersReply, error) {
	var data any
	var err error

	if len(in.GetIds()) > 0 {
		data, err = services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
			users, err := queries.GetUsersByIds(tx, ctx, utils.Int32SliceToIntSlice(in.GetIds()), int(in.Limit), int(in.Offset))
			return users, err
		})()
	} else {
		data, err = services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
			users, err := queries.GetUsers(tx, ctx, int(in.Limit), int(in.Offset))
			return users, err
		})()
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return &profiles.GetUsersReply{}, nil
		} else {
			return &profiles.GetUsersReply{}, err
		}
	}

	users, ok := data.([]entities.User)
	if !ok {
		log.Error("Unable to get to users", api.ERROR_ASSERT_RESULT_TYPE)
		return &profiles.GetUsersReply{}, err
	}

	return &profiles.GetUsersReply{Users: toGetUsersReply(users)}, nil
}

func (s *ProfilesServiceServer) GetUsersStream(stream profiles.ProfilesService_GetUsersStreamServer) error {
	return fmt.Errorf("NOT IMPLEMENTED") // TODO
}

func toGetUserReply(user *entities.User) *profiles.GetUserReply {
	return &profiles.GetUserReply{
		Id:             int32(user.Id),
		Login:          user.Login,
		Email:          user.Email,
		Role:           user.Role,
		State:          user.State,
		CreateDate:     timestamppb.New(user.CreateDate),
		LastUpdateDate: timestamppb.New(user.LastUpdateDate),
	}
}

func toGetUsersReply(users []entities.User) []*profiles.GetUserReply {
	var result []*profiles.GetUserReply = make([]*profiles.GetUserReply, 0, len(users))
	for _, u := range users {
		result = append(result, toGetUserReply(&u))
	}
	return result
}
