package validator

import (
	"backend/model"

	"github.com/Oudwins/zog"
)

var UserIdShape = zog.Shape{
	"UserID": zog.Int64().Required().GT(0),
}

var BaseShape = zog.Shape{
	"Email": zog.String().Email().Required(),
}

var PasswordShape = zog.Shape{
	"Password": zog.String().Min(8).Required(),
}

var ConfirmShape = zog.Shape{
	"ConfirmPassword": zog.String().Required(),
}

func PasswordMatchTest(dataPtr any, ctx zog.Ctx) bool {
	matcher, ok := dataPtr.(model.PasswordMatcher)
	if !ok {
		return true
	}

	if matcher.GetPassword() != matcher.GetConfirm() {
		ctx.AddIssue(&zog.ZogIssue{
			Path:    []string{"confirmPassword"},
			Message: "Passwords do not match",
		})
		return false
	}
	return true
}
