package controller

import (
	"backend/model"
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// RegisterRoutes sets up the health check endpoint.
func (ctrl *HealthController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/api/health",
		Summary:     "System Health Check",
		Description: "Confirm that the server is up and running. Used by Load Balancers and Uptime Monitors.",
		Tags:        []string{"System"},
	}, ctrl.healthCheck)
}

func (ctrl *HealthController) healthCheck(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	return NewResponse(map[string]string{"status": "UP"}, "System is operational"), nil
}
