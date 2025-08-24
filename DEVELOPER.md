# Developer Documentation

### AI Notes

- Prefer standard-library over 3rd party dependencies
- Don't implement entire functions without being asked first

### Data Structure

The main thread finds markdown files, parses them for frontmatter and writes them back if there are pending changes.

- MDFile is a parsed markdown file that we can write back if needed.
- Publishers.Publish(mdf) take a MDFile reference, publish and register themselves back
- OG-Image generator checks for `ogImage` and will create a `<slug>.og.svg` if it's missing.
