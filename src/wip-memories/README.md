Just for jotting down memory titles and song ids for each section.
We can then use xargs to parse these through to the song fetcher to write the memory.
```
-n <name>
-songids <ids> 
--othersongids <ids>
```

Running it through the songfetcher:
```sh
build/songfetcher $(xargs -a src/wip-memories/filename)
```