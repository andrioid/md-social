package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	appbsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/urfave/cli/v3"
)

type BlueskyPublisher struct {
	Handle      string
	AppPassword string
	Host        string
	Client      *xrpc.Client
	ctx         context.Context
}

// Creates a publisher, or exits if it fals
func NewBluesky(ctx context.Context, cmd *cli.Command) (*BlueskyPublisher, error) {

	bsk := &BlueskyPublisher{
		Handle:      cmd.String("bluesky-handle"),
		AppPassword: cmd.String("bluesky-app-pw"),
		Host:        cmd.String("bluesky-host"),
		ctx:         ctx,
	}
	if bsk.Handle == "" {
		return nil, fmt.Errorf("%w: no bluesky handle defined, skipping", ErrModuleSkipped)
	}
	if bsk.AppPassword == "" {
		return nil, fmt.Errorf("bluesky app password required")
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

	return bsk, nil
}

func (bsk *BlueskyPublisher) PublisherID() string {
	return "bluesky"
}

func (bsk *BlueskyPublisher) Publish(ctx context.Context, md *MDFile) error {
	if dryRun {
		fmt.Println("[bluesky.publish] dry-run", md.Filename)
		return nil
	} else {
		fmt.Println("[bluesky.publish] posting", md.Filename)
	}
	title := md.Title()
	mdURL := md.URL()

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
		return err
	}
	md.SetSocial(bsk.PublisherID(), res.Uri)
	return nil
}
