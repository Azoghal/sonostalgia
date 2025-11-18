package sonostalgia

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Memory struct {
	PageTitle  string `yaml:"shortTitle"`
	Title      string `yaml:"title"`
	Subtitle   string `yaml:"subtitle"`
	Date       string `yaml:"date"`
	Songs      []Song `yaml:"songs"`
	Content    string `yaml:"content"` // load strings from file, we convert markdown to html in the template
	OtherSongs []Song `yaml:"otherSongs"`
}

type Song struct {
	Name         string `yaml:"name"`
	SongLink     string `yaml:"link"`
	Artist       string `yaml:"artist"`
	ArtistLink   string `yaml:"artistLink"`
	RelevantDate string `yaml:"relevantDate"` // string as it's free-form
	// SpotifyId string - could use this to populate the above for each song rather than having to manaully find them all
}

func LoadMemory(filename string) (*Memory, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var memory Memory
	err = yaml.Unmarshal(data, &memory)
	if err != nil {
		return nil, err
	}

	return &memory, nil
}
