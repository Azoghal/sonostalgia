package sonostalgia

type About struct {
	// Body
	EarliestMemory string // date of earliest memory
	MemoryCount    int
	SongCount      int
	ArtistCount    int
}

var (
	ExampleAbout = About{
		EarliestMemory: "2017",
		MemoryCount:    1,
		SongCount:      1,
		ArtistCount:    1,
	}
)
