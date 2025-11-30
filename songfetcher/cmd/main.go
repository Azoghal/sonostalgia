package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"unicode"

	"github.com/alexflint/go-arg"
	sonostalgia "github.com/azoghal/sonostalgia/src"
	"github.com/joho/godotenv"
	spotify "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

/* fetches all the info to populate a song in order to populate song(s) in memories

- name:
  link:
  artist:
  artistLink:
  relevantDate:
  imageLink:

*/

const (
	minDesiredWidth = 100 // if we can, make sure all images are at least 100px width/height
	maxDesiredWidth = 350 // if we can, try to keep the images a reasonable size
)

type Args struct {
	MemoryOutputTitle string   `arg:"-n,--name,required"      help:"the output name of the memory, e.g. eve-online"`
	SongIds           []string `arg:"--songids,required"      help:"list of spotify ids for the main songs"`
	OtherSongIds      []string `arg:"--othersongids" help:"list of spotify ids for the other songs"`
}

type TemplateParams struct {
	OutputTitle string
	Songs       []sonostalgia.Song
	OtherSongs  []sonostalgia.Song
}

// Attempt to open the output file
// load the memory template
// load the env vars
// get a spotify api client
// lookup songs
// lookup otherSongs
// do template
// write to file
func main() {
	var args Args
	arg.MustParse(&args)

	fileWriter, err := os.Create(fmt.Sprintf("songfetcher/output/%s.yaml", args.MemoryOutputTitle))
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer fileWriter.Close()

	memoryTemplate, err := template.ParseFiles("songfetcher/templates/memory.template.yaml")
	if err != nil {
		log.Fatal("Error parsing template: ", err)
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatal("failed to load env file")
	}

	spotifyClientId := os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     spotifyClientId,
		ClientSecret: spotifyClientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		log.Fatalf("couldn't get token: %v", err)
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)

	fmt.Println()

	songs := []sonostalgia.Song{}
	otherSongs := []sonostalgia.Song{}

	for _, songId := range args.SongIds {
		song, err := lookupSongById(ctx, client, songId)
		if err != nil {
			log.Printf("FAILED to lookup song: %s\n", err)
			continue
		}

		songs = append(songs, *song)
	}

	for _, songId := range args.OtherSongIds {
		song, err := lookupSongById(ctx, client, songId)
		if err != nil {
			log.Printf("FAILED to lookup song: %s\n", err)
		}

		otherSongs = append(otherSongs, *song)
	}

	templateParams := TemplateParams{
		OutputTitle: args.MemoryOutputTitle,
		Songs:       songs,
		OtherSongs:  otherSongs,
	}

	err = memoryTemplate.Execute(fileWriter, templateParams)
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}
}

func lookupSongById(ctx context.Context, client *spotify.Client, id string) (*sonostalgia.Song, error) {

	testId := spotify.ID(id)

	track, err := client.GetTrack(ctx, testId, spotify.Market("GB"))
	if err != nil {
		return nil, errors.New("track request failed")
	}

	album, err := client.GetAlbum(ctx, track.Album.ID, spotify.Market("GB"))
	if err != nil {
		return nil, errors.New("album request failed")
	}

	artists := []sonostalgia.Artist{}
	for _, artist := range track.Artists {
		artists = append(artists, sonostalgia.Artist{
			Name: artist.Name,
			Link: artist.ExternalURLs["spotify"],
		})
	}

	bestImageAssetUrl := fetchBestImage(album.Images, "src/assets", makeImageName(track.Name))

	// RelevantDate left empty as it needs user input
	song := &sonostalgia.Song{
		Name:      track.Name,
		Artists:   artists,
		SongLink:  track.ExternalURLs["spotify"],
		ImageLink: bestImageAssetUrl,
	}

	return song, nil
}

func makeImageName(trackName string) string {
	alphaOnly := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, trackName)

	lowerAlphaOnly := strings.ToLower(alphaOnly)

	return strings.Join(strings.Fields(lowerAlphaOnly), "-")
}

// fetchBestImage finds and downloads the image which matches the most constraints.
// it will name the downloaded artefact outputName.<ext> where ext = jpg,png
// if there are no images, the empty string will be returned
func fetchBestImage(images []spotify.Image, outputDir string, outputName string) string {

	var image *spotify.Image = nil
	bestScore := 0

	for _, img := range images {
		width := int(img.Width)
		score := 1
		if width < maxDesiredWidth {
			score += 1
		}
		if width > minDesiredWidth {
			score += 1
		}
		if score > bestScore {
			image = &img
			bestScore = score
		}
	}

	// download
	outputFilename := fmt.Sprintf("%s/%s.jpg", outputDir, outputName)
	err := downloadImage(image.URL, outputFilename)
	if err != nil {
		log.Printf("failed to download image: %v", err)
	}

	return fmt.Sprintf("assets/%s.jpg", outputName)
}

func downloadImage(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
