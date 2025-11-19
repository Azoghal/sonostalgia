# Memory files

Memory files contains metadata and content constituting a "memory" as required by the sononstalgia static site generator. They are `.yaml` or `.yml` files that conform to the following format:

## Format
```yaml
shortTitle: Page Title Here
title: Main Title
subtitle: Optional Subtitle
date: "2025-01-15"

songs:
  - name: Song Title
    link: https://open.spotify.com/track/...
    artist: Artist Name
    artistLink: https://open.spotify.com/artist/...
    relevantDate: Summer 2024

content: |
  # Main Content
  
  This is the **markdown content** that will be converted to HTML.
  
  ## Section
  
  - Point one
  - Point two
  
  You can include [links](https://example.com) and other markdown features.

otherSongs:
  - name: Related Song
    link: https://spotify.com/...
    artist: Artist Name
    artistLink: https://spotify.com/artist/...
    relevantDate: "2022"
```