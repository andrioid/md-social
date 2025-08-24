package main

import "context"

type MDFProcessor interface {
	// Processes a markdown document, or fail trying
	Process(ctx context.Context, mdf *MDFile) error
}
