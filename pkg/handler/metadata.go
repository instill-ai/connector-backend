package handler

import (
	"context"
	"strings"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func extractFromMetadata(ctx context.Context, key string) ([]string, bool) {
	data, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return []string{}, false
	}
	return data[strings.ToLower(key)], true
}

func getResourceNameID(name string) (string, error) {
	id := name[strings.LastIndex(name, "/")+1:]
	if id == "" {
		return "", status.Errorf(codes.InvalidArgument, "Error when extract resource id from resource name `%s`", name)
	}
	return id, nil
}

func getResourcePermalinkUID(permalink string) (string, error) {
	uid := permalink[strings.LastIndex(permalink, "/")+1:]
	if uid == "" {
		return "", status.Errorf(codes.InvalidArgument, "Error when extract resource id from resource permalink `%s`", permalink)
	}
	return uid, nil
}

func getOwner(ctx context.Context) (string, error) {
	metadatas, ok := extractFromMetadata(ctx, "owner")
	if ok {
		if len(metadatas) == 0 {
			return "", status.Error(codes.InvalidArgument, "Cannot find `owner` in your request")
		}
		return metadatas[0], nil
	}
	return "", status.Error(codes.InvalidArgument, "Error when extract `owner` metadata")
}
