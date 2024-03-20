package profiles

import (
	"context"
	"database/sql"

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
