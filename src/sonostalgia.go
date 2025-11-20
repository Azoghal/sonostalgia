package sonostalgia

import "fmt"

// All the parsed template params
type Sonostalgia struct {
	IndexParams    Index
	AboutParams    About
	MemoriesParams Memories
	YearsParams    Years
	MemoryParams   []Memory
}

func LoadSonostalgia(memoryFiles []string) (*Sonostalgia, error) {
	var memories []Memory
	for _, file := range memoryFiles {
		memory, err := LoadMemory(file)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", file, err)
		}
		memories = append(memories, *memory)
	}

	return &Sonostalgia{
		MemoryParams: memories,
	}, nil
}
