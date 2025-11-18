package sonostalgia

type Index struct {
	// Body
	MemoryCount      int
	SongCount        int
	ArtistCount      int
	YearsWithEntries int

	RecentMemories []Memory
}
