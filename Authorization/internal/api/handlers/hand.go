// Package handlers
package handlers

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/api/service"
	"github.com/Wladim1r/auth/lib/errs"
	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/Wladim1r/auth/lib/hashpwd"
	"github.com/Wladim1r/proto-crypto/gen/protos/auth-portfile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type handler struct {
	auth.UnimplementedAuthServer
	us service.UserService
	ts service.TokenService
	ur repository.UserRepository
}

func RegisterServer(
	gRPC *grpc.Server,
	uServ service.UserService,
	tServ service.TokenService,
	uRepo repository.UserRepository,
) {
	auth.RegisterAuthServer(gRPC, &handler{
		us: uServ,
		ts: tServ,
		ur: uRepo,
	})
}

func (h *handler) Register(ctx context.Context, req *auth.AuthRequest) (*auth.Empty, error) {
	name := req.GetName()
	password := req.GetPassword()

	err := h.us.CheckUserExistsByName(name)

	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			hashPwd, err := hashpwd.HashPwd([]byte(password))
			if err != nil {
				return nil, status.Error(codes.Internal, "Could not hash password: "+err.Error())
			}

			if err := h.us.CreateUser(name, hashPwd); err != nil {
				switch {
				case errors.Is(err, errs.ErrRecordingWNC):
					return nil, status.Error(
						codes.Internal,
						"Could not create user rawsAffected=0: "+err.Error(),
					)
				default:
					return nil, status.Error(codes.Internal, "Could not create user: "+err.Error())
				}
			}
			return &auth.Empty{}, nil

		default:
			return nil, status.Error(codes.Internal, "db error: "+err.Error())
		}
	}

	return nil, status.Error(codes.AlreadyExists, "User has already existed 💩")
}

func (h *handler) Login(ctx context.Context, req *auth.AuthRequest) (*auth.TokenResponse, error) {
	name := req.GetName()
	password := req.GetPassword()

	userID, err := checkUserExists(name, password, h.ur)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			return nil, status.Error(codes.NotFound, "User not found")
		case errors.Is(err, errs.ErrDB):
			return nil, status.Error(codes.Internal, "db error: "+err.Error())
		default:
			return nil, status.Error(codes.Internal, "unknown error: "+err.Error())
		}
	}

	refreshTTL := getenv.GetTime("REFRESH_TTL", 150*time.Second)

	access, refresh, err := h.ts.SaveToken(userID, time.Now().Add(refreshTTL))
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			return nil, status.Error(codes.NotFound, "could not found user: "+err.Error())

		case errors.Is(err, errs.ErrRecordingWNC):
			return nil, status.Error(codes.Internal, "could not create token: "+err.Error())
		case errors.Is(err, errs.ErrDB):
			return nil, status.Error(codes.Internal, "db error: "+err.Error())

		case errors.Is(err, errs.ErrSignToken):
			return nil, status.Error(codes.Internal, "jwt error: "+err.Error())

		default:
			return nil, status.Error(codes.Internal, "unknown error: "+err.Error())
		}
	}

	userIDstr := strconv.Itoa(int(userID))

	userIDHeader := metadata.Pairs(
		"x-user-id", userIDstr,
	)
	grpc.SendHeader(ctx, userIDHeader)

	return &auth.TokenResponse{Access: access, Refresh: refresh}, nil
}

func (h *handler) Refresh(
	ctx context.Context,
	req *auth.Empty,
) (*auth.TokenResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "could not found metadata in grpc context")
	}

	userIDstr := md.Get("x-user-id")
	userID, err := strconv.Atoi(userIDstr[0])
	if err != nil {
		panic(err)
	}

	refreshToken := md.Get("x-refresh-token")[0]

	refreshTTL := getenv.GetTime("REFRESH_TTL", 150*time.Second)

	access, refresh, err := h.ts.RefreshAccessToken(
		refreshToken,
		userID,
		time.Now().Add(refreshTTL),
	)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrTokenTTL):
			return nil, status.Error(codes.PermissionDenied, err.Error())

		case errors.Is(err, errs.ErrRecordingWNF):
			return nil, status.Error(codes.NotFound, "could not found user: "+err.Error())

		case errors.Is(err, errs.ErrRecordingWNC):
			return nil, status.Error(codes.Internal, "could not create token: "+err.Error())

		case errors.Is(err, errs.ErrDB):
			return nil, status.Error(codes.Internal, "db error: "+err.Error())
		case errors.Is(err, errs.ErrSignToken):
			return nil, status.Error(codes.Internal, "jwt error: "+err.Error())

		default:
			return nil, status.Error(codes.Internal, "unknown error: "+err.Error())
		}
	}

	return &auth.TokenResponse{
		Access:  access,
		Refresh: refresh,
	}, nil
}

func (h *handler) Logout(ctx context.Context, req *auth.Empty) (*auth.Empty, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "could not found metadata in grpc context")
	}

	refreshToken := md.Get("x-refresh-token")[0]
	authHeader := md.Get("x-authorization-header")[0]

	if err := checkAuth(authHeader, h.ur); err != nil {
		switch {
		case errors.Is(err, errs.ErrEmptyAuthHeader):
			return nil, status.Error(
				codes.Unauthenticated,
				"user is not registered: "+err.Error(),
			)
		case errors.Is(err, errs.ErrInvalidToken):
			return nil, status.Error(codes.PermissionDenied, err.Error())

		case errors.Is(err, errs.ErrTokenTTL):
			return nil, status.Error(codes.PermissionDenied, err.Error())

		case errors.Is(err, errs.ErrRecordingWNF):
			return nil, status.Error(codes.NotFound, "user does not exist: "+err.Error())

		case errors.Is(err, errs.ErrDB):
			return nil, status.Error(codes.Internal, "db error: "+err.Error())

		default:
			return nil, status.Error(codes.Internal, "unknowm error: "+err.Error())
		}
	}

	if err := h.ts.DeleteToken(refreshToken); err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			return nil, status.Error(codes.NotFound, "token not found :"+err.Error())

		case errors.Is(err, errs.ErrDB):
			return nil, status.Error(codes.Internal, "db error: "+err.Error())

		default:
			return nil, status.Error(codes.Internal, "unknown error: "+err.Error())
		}
	}

	return &auth.Empty{}, nil
}
