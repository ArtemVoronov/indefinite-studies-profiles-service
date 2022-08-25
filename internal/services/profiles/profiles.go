package profiles

import (
	"context"

	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/services/credentials"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/profiles"
	"google.golang.org/grpc"
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

	return &profiles.ValidateCredentialsReply{UserId: int32(result.UserId), IsValid: result.IsValid}, nil
}
