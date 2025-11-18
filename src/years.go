package sonostalgia

type Years struct {
	// Body
	Years []Year
}

type Year struct {
	Year     string
	Memories []Memory
}
