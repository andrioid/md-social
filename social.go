package main

import "context"

type Publisher interface {
	Publish(ctx context.Context, md *MDFile) (string, error)
	PublisherID() string
}
