package sonostalgia

type Memory struct {
	// Head
	PageTitle string

	// Body
	Title    string
	Subtitle string
	Date     string

	// Songs
	Songs []Song

	// Content / Memory
	Content string // string for now - might want to work out what to do to support markdown?

	// Other Songs
	OtherSongs []Song
}

type Song struct {
	Name         string
	SongLink     string
	Artist       string
	ArtistLink   string
	RelevantDate string // string as it's free-form
}

var (
	ExampleMemory = Memory{
		PageTitle: "Ageas Bowl 1:1s",
		Title:     "Cricket 1:1s with Sam at the Ageas bowl",
		Date:      "2016-2019",
		Songs: []Song{
			{
				Name:         "Pristine",
				SongLink:     "",
				Artist:       "Mantaraybryn",
				ArtistLink:   "",
				RelevantDate: "March 2017",
			},
		},
		Content: `Sol driving me to the ageas bowl for my 1:1 with Sam. 
He'd just found Pristine by Mantaraybryn and we played it in the car a couple of times.
I visualise the road in hedge end alongside KFC/Pizza hut, just before the roundabout where you turn left for the ageas bowl.

Going to pizza hut with Dad before/after sessions. Eating a huge amount of pizza and a slab of cheesecake after doing cross country, then Sam made me run between the wickets for the first time that year!

Having too much pizza and pepsi at pizza hut, feeling like I was going to explode on the way home in the car.

The bowling machine with LED screen. The spin bowling machine.

Speedy trips back along the M27 and A31, radio 4.`,
		OtherSongs: []Song{},
	}
)
