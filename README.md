# Markdown Social

Parses Markdown articles and posts them automatically on social-media. It also stores reference to the post so we don't create duplicates.

## Requirements

### Frontmatter

- `title`: string
- `date`: RFC 3339
- `social`: map[string]string (e.g. {"bluesky": "at://asdf"})

## Usage

md-social blogdir/

## Configuration

All configuration is via ENV variables. Providers without configuration are skipped

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
