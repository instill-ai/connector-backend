package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
)

func (s *service) ownerRscNameToPermalink(ownerRscName string) (ownerPermalink string, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if strings.Split(ownerRscName, "/")[0] == "users" {
		user, err := s.userServiceClient.GetUser(ctx, &mgmtPB.GetUserRequest{Name: ownerRscName})
		if err != nil {
			return "", fmt.Errorf("[mgmt-backend] %s", err)
		}
		ownerPermalink = "users/" + user.User.GetUid()
	} else if strings.Split(ownerRscName, "/")[0] == "orgs" { //nolint
		// TODO: implement orgs case
	}

	return ownerPermalink, nil
}
