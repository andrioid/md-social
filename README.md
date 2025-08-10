# Markdown Social

Parses Markdown articles and posts them automatically on social-media. It also stores reference to the post so we don't create duplicates.

The idea is to run this via build-pipelines like so:

1. Site is built and deployed
2. This program runs and posts what needs posting
3. If there were changes in Git, it should commit them and rebuild the site
4. Now possible to use `social.bluesky` (or whatever) to setup likes/comments

## Requirements

### Frontmatter

- `title`: string
- `date`: RFC 3339
- `social`: map[string]string (e.g. {"bluesky": "at://asdf"})

## Usage

md-social blogdir/

## Configuration

All configuration is via ENV variables. Providers without configuration are skipped

- [ ] MD_BASE_URL

### Bluesky

- BLUESKY_DID
- BLUESKY_APP_PASSWORD

## Tasks

As a CLI user, I can...

- [x] Parse one or many markdown files for frontmatter
- [x] Persist the post URL in the markdown file
- [x] Post to Bluesky
- [x] Skip if already added
- [x] Handle single file
- [x] Determine if we should create a post from `date` frontmatter field
- [ ] Handle directory and strip the prefix
- [ ] Build and publish to NPM (bin) so we can use it in our Astro project

## Maybe later

- LinkedIn
- ActivityPub
- dev.to
