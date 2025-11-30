# Memory files

Memory files contains metadata and content constituting a "memory" as required by the sononstalgia static site generator. They are `.yaml` or `.yml` files that conform to the following format:

## Format
```yaml
outputTitle: Output Filename here (<bob> => bob.html)
shortTitle: Page Title Here
title: Main Title
subtitle: Optional Subtitle
date: "2025-01-15"

songs:
  - name: Song Title
    link: https://open.spotify.com/track/...
    artists:
      - name: Artist Name
        link: https://open.spotify.com/artist/...
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
    artists:
      - name: Artist Name
        link: https://open.spotify.com/artist/...
    relevantDate: "2022"
```

## Generation

You can more quickly generate these files by using the songfetcher program in this repo. It takes an output file name, list of song ids and list of other song ids, and will produce a prepopulated memory file. This can then be edited as desired. Separating this out from the actual templating process means there's still complete flexibility when it comes to building the website, i.e. we're not tied to a particular music platform like Spotify, which is what the songfetcher uses.