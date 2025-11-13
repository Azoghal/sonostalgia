package sonostalgia

type Memories struct {
	AllMemories []Memory
}

var (
	ExampleMemories = Memories{
		AllMemories: []Memory{ExampleMemory},
	}
)
