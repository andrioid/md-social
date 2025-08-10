package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	appbsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
)

type BlueskyPublisher struct {
	Handle      string
	AppPassword string
	Host        string
	Client      *xrpc.Client
	ctx         context.Context
}

// Creates a publisher, or exits if it fals
func NewBluesky(ctx context.Context) *BlueskyPublisher {
	bsk := &BlueskyPublisher{
		Handle:      os.Getenv("BLUESKY_HANDLE"),
		AppPassword: os.Getenv("BLUESKY_APP_PASSWORD"),
		Host:        os.Getenv("BLUESKY_HOST"),
		ctx:         ctx,
	}
	if bsk.Handle == "" {
		log.Fatal("BLUESKY_HANDLE required")
	}
	if bsk.AppPassword == "" {
		log.Fatal("BLUESKY_APP_PASSWORD required")
	}
	if bsk.Host == "" {
		bsk.Host = "https://bsky.social"
	}

	bsk.Client = &xrpc.Client{
		Client: new(http.Client),
		Host:   bsk.Host,
	}

	session, err := atproto.ServerCreateSession(ctx, bsk.Client, &atproto.ServerCreateSession_Input{
		Identifier: bsk.Handle,
		Password:   bsk.AppPassword,
	})
	if err != nil {
		log.Fatal("Failed to initialise Bluesky", err)
	}
	bsk.Client.Auth = &xrpc.AuthInfo{
		AccessJwt:  session.AccessJwt,
		RefreshJwt: session.RefreshJwt,
		Handle:     session.Handle,
		Did:        session.Did,
	}

	return bsk
}

func (bsk *BlueskyPublisher) PublisherID() string {
	return "bluesky"
}

func (bsk *BlueskyPublisher) Publish(ctx context.Context, md *MDFile) (string, error) {
	p := md.GetPost()
	title := p.title
	mdURL := p.url

	post := appbsky.FeedPost{
		Text:          title,
		LexiconTypeID: "app.bsky.feed.post",
		CreatedAt:     time.Now().Format(time.RFC3339),
		Embed: &appbsky.FeedPost_Embed{
			EmbedExternal: &appbsky.EmbedExternal{
				LexiconTypeID: "app.bsky.embed.external",
				External: &appbsky.EmbedExternal_External{
					Title: title,
					Uri:   mdURL,
				},
			},
		},
	}
	postInput := &atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       bsk.Client.Auth.Did,
		Record:     &lexutil.LexiconTypeDecoder{Val: &post},
	}

	res, err := atproto.RepoCreateRecord(bsk.ctx, bsk.Client, postInput)
	if err != nil {
		return "", err
	}
	return res.Uri, nil
}
