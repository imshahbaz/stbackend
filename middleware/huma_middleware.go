package middleware

import (
	"backend/auth"
	"backend/model"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func HumaAuthMiddleware(api huma.API, isProduction bool) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		cookieHeader := ctx.Header("Cookie")
		token := ""

		parts := strings.SplitSeq(cookieHeader, ";")
		for part := range parts {
			part = strings.TrimSpace(part)
			if after, ok := strings.CutPrefix(part, "auth_token="); ok {
				token = after
				break
			}
		}

		if token == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")
			return
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid session")
			return
		}

		if time.Until(claims.ExpiresAt.Time) < 15*time.Minute {
			newToken, _ := auth.GenerateToken(claims.User)
			cookie := http.Cookie{
				Name:     "auth_token",
				Value:    newToken,
				Path:     "/",
				MaxAge:   1800,
				Secure:   isProduction,
				HttpOnly: true,
			}
			ctx.SetHeader("Set-Cookie", cookie.String())
		}

		ctx = huma.WithValue(ctx, "user", claims.User)
		next(ctx)
	}
}

func HumaAdminOnly(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		userVal := ctx.Context().Value("user")
		user, ok := userVal.(model.UserDto)
		if !ok || user.Role != model.RoleAdmin {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: Admin access required")
			return
		}
		next(ctx)
	}
}
