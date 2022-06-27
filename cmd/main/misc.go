package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogo/status"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

func httpResponseModifier(ctx context.Context, w http.ResponseWriter, p proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	// set http status code
	if vals := md.HeaderMD.Get("x-http-code"); len(vals) > 0 {
		code, err := strconv.Atoi(vals[0])
		if err != nil {
			return err
		}
		// delete the headers to not expose any grpc-metadata in http response
		delete(md.HeaderMD, "x-http-code")
		delete(w.Header(), "Grpc-Metadata-X-Http-Code")
		w.WriteHeader(code)
	}

	return nil
}

func handleError(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	if s, ok := status.FromError(err); ok {
		switch {
		case s.Code() == codes.FailedPrecondition && strings.Contains(s.Message(), "[DELETE]"):
			errorResponse(w,
				http.StatusUnprocessableEntity,
				http.StatusText(http.StatusUnprocessableEntity),
				s.Message(),
			)
		case s.Code() == codes.FailedPrecondition:
			errorResponse(w,
				http.StatusPreconditionFailed,
				http.StatusText(http.StatusPreconditionFailed),
				s.Message(),
			)
		case s.Code() == codes.InvalidArgument:
			errorResponse(w,
				http.StatusBadRequest,
				http.StatusText(http.StatusBadRequest),
				s.Message(),
			)
		case s.Code() == codes.AlreadyExists:
			errorResponse(w,
				http.StatusConflict,
				http.StatusText(http.StatusConflict),
				s.Message(),
			)
		}
	}
}

func customMatcher(key string) (string, bool) {
	if strings.HasPrefix(strings.ToLower(key), "jwt-") {
		return key, true
	}

	switch key {
	case "owner":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func errorResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(datamodel.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}
