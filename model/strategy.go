package model

import "strings"

// --- STRATEGY ---
// Strategy is the core scanner entity
type Strategy struct {
	Name       string `bson:"_id" json:"name"`
	ScanClause string `bson:"scanClause" json:"scanClause"`
	Active     bool   `bson:"active" json:"active"`
}

// StrategyDto is used for creating/updating strategies
type StrategyDto struct {
	Name       string `json:"name" validate:"required"`
	ScanClause string `json:"scanClause" validate:"required"`
	Active     bool   `json:"active"`
}

func (d *StrategyDto) ToEntity() Strategy {
	return Strategy{
		Name:       strings.ToUpper(d.Name),
		ScanClause: d.ScanClause,
		Active:     d.Active,
	}
}

// --- Huma Structs ---

type CreateStrategyRequest struct {
	Body StrategyDto
}

type DeleteStrategyInput struct {
	ID string `query:"id" doc:"Strategy ID (Name)"`
}
