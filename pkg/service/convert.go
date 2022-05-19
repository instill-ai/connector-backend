package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	mgmtPB "github.com/instill-ai/protogen-go/mgmt/v1alpha"
)

func (s *service) ownerNameToPermalink(owner *string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if strings.Split(*owner, "/")[0] == "users" {
		user, err := s.userServiceClient.GetUser(ctx, &mgmtPB.GetUserRequest{Name: *owner})
		if err != nil {
			return fmt.Errorf("[mgmt-backend] %s", err)
		}
		*owner = "users/" + user.User.GetUid()
	} else if strings.Split(*owner, "/")[0] == "orgs" { //nolint
		// TODO: implement orgs case
	}

	return nil
}

func (s *service) ownerPermalinkToName(owner *string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if strings.Split(*owner, "/")[0] == "users" {
		user, err := s.userServiceClient.LookUpUser(ctx, &mgmtPB.LookUpUserRequest{Permalink: *owner})
		if err != nil {
			return fmt.Errorf("[mgmt-backend] %s", err)
		}
		*owner = "users/" + user.User.GetId()
	} else if strings.Split(*owner, "/")[0] == "orgs" { //nolint
		// TODO: implement orgs case
	}

	return nil
}
