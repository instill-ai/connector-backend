package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/x/sterr"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
)

func (s *service) ownerRscNameToPermalink(ownerRscName string) (ownerPermalink string, err error) {

	logger, _ := logger.GetZapLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if strings.Split(ownerRscName, "/")[0] == "users" {
		user, err := s.mgmtAdminServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{Name: ownerRscName})
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				"[mgmt-backend] get user error",
				"users",
				ownerRscName,
				"",
				fmt.Sprintf("[mgmt-backend]: %s", err.Error()),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return "", st.Err()
		}
		ownerPermalink = "users/" + user.User.GetUid()
	} else if strings.Split(ownerRscName, "/")[0] == "orgs" { //nolint
		// TODO: implement orgs case
	}

	return ownerPermalink, nil
}
