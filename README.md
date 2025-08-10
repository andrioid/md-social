# Markdown Social

Parses Markdown articles and posts them automatically on social-media. It also stores reference to the post so we don't create duplicates.

## Usage

md-social blog/\*.md

## Configuration

### Bluesky

- BLUESKY_DID
- BLUESKY_APP_PASSWORD

## Tasks

As a CLI user, I can...

- [ ] Parse one or many markdown files for frontmatter
- [ ] Determine if we should create a post from `date` frontmatter field, or modification date
- [ ] Mock post creation
- [ ] Persist the post URL in the markdown file
