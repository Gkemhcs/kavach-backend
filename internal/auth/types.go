package auth

import (
	"context"

	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
)

type UserInfoGetter interface {
	GetUserInfoByGithubUserName(ctx context.Context, userName string) (*userdb.User, error)
}
