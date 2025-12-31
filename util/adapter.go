package util

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func BindDynamic(ctx context.Context, target interface{}) error {
	hctx, ok := ctx.(huma.Context)
	if !ok {
		return fmt.Errorf("context is not a huma.Context")
	}

	r, _ := humago.Unwrap(hctx)
	if r == nil {
		return fmt.Errorf("could not unwrap request")
	}

	return json.NewDecoder(r.Body).Decode(target)
}
