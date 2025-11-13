package sonostalgia

type Index struct {
	// Body
	MemoryCount      int
	SongCount        int
	ArtistCount      int
	YearsWithEntries int

	RecentMemories []Memory
}

var (
	ExampleIndex = Index{
		MemoryCount:      1,
		SongCount:        1,
		ArtistCount:      1,
		YearsWithEntries: 1,

		RecentMemories: []Memory{ExampleMemory},
	}
)
