package main

import (
	"context"
)

type Publisher interface {
	Publish(ctx context.Context, md *MDFile) error
	PublisherID() string
}

type Processor interface {
	Process(ctx context.Context, mdf *MDFile) error
}
