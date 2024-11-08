package app

import "context"

type Applacation struct {
}

func New(ctx context.Context) (*Applacation, error) {
	return &Applacation{}, nil
}
