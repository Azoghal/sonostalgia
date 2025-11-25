package sonostalgia

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Memory struct {
	OutputTitle string `yaml:"outputTitle"` // filename
	PageTitle   string `yaml:"shortTitle"`
	Title       string `yaml:"title"`
	Subtitle    string `yaml:"subtitle"`
	Date        string `yaml:"date"`
	Songs       []Song `yaml:"songs"`
	Content     string `yaml:"content"` // load strings from file, we convert markdown to html in the template
	OtherSongs  []Song `yaml:"otherSongs"`
}

type Song struct {
	Name         string `yaml:"name"`
	SongLink     string `yaml:"link"`
	Artist       string `yaml:"artist"`
	ArtistLink   string `yaml:"artistLink"`
	RelevantDate string `yaml:"relevantDate"` // string as it's free-form
	ImageLink    string `yaml:"imageLink"`
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

func (s Song) String() string {
	return fmt.Sprintf(`
- name: %s
  link: %s
  artist: %s 
  artistLink: %s
  imageLink: %s
  relevantDate: %s`, s.Name, s.SongLink, s.Artist, s.ArtistLink, s.ImageLink, s.RelevantDate)
}
