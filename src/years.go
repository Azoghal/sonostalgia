package sonostalgia

type Years struct {
	// Body
	Years []Year
}

type Year struct {
	Year     string
	Memories []Memory
}

var (
	ExampleYears = Years{
		Years: []Year{
			{
				Year: "2017",
				Memories: []Memory{
					ExampleMemory,
				},
			},
		},
	}
)
