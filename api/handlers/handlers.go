package handlers

import "github.com/abdotop/wave-pool/db/sqlc"

type API struct {
	db sqlc.Querier
}

func NewAPI(db sqlc.Querier) *API {
	return &API{db: db}
}
