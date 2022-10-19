package profiles

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/profiles"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

	return &profiles.ValidateCredentialsReply{UserUuid: result.UserUuid, IsValid: result.IsValid, Role: result.Role}, nil
}

func (s *ProfilesServiceServer) GetUser(ctx context.Context, in *profiles.GetUserRequest) (*profiles.GetUserReply, error) {
	user, err := services.Instance().Profiles().GetUser(in.Uuid)

	if err != nil {
		if err == sql.ErrNoRows {
			return &profiles.GetUserReply{}, nil
		} else {
			return &profiles.GetUserReply{}, err
		}
	}

	return toGetUserReply(&user), nil
}

func (s *ProfilesServiceServer) GetUsers(ctx context.Context, in *profiles.GetUsersRequest) (*profiles.GetUsersReply, error) {
	users, err := services.Instance().Profiles().GetUsers(int(in.GetOffset()), int(in.GetLimit()), int(in.GetShard()))
	if err != nil {
		if err == sql.ErrNoRows {
			return &profiles.GetUsersReply{}, nil
		} else {
			return &profiles.GetUsersReply{}, err
		}
	}
	result := &profiles.GetUsersReply{
		Offset:      in.Offset,
		Limit:       in.Limit,
		Count:       int32(len(users)),
		Users:       toGetUsersReply(users),
		ShardsCount: int32(services.Instance().Profiles().ShardsNum),
	}

	return result, nil
}

func (s *ProfilesServiceServer) GetUsersStream(stream profiles.ProfilesService_GetUsersStreamServer) error {
	return fmt.Errorf("NOT IMPLEMENTED") // TODO
}

func toGetUserReply(user *entities.User) *profiles.GetUserReply {
	return &profiles.GetUserReply{
		Id:             int32(user.Id),
		Uuid:           user.Uuid,
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
