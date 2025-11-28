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
	SongIds      []string `arg:"--songids"      help:"list of spotify ids for the main songs"`
	OtherSongIds []string `arg:"--othersongids" help:"list of spotify ids for the other songs"`
}

func main() {
	var args Args
	arg.MustParse(&args)

	err := godotenv.Load()
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

	for _, songId := range args.SongIds {
		song, err := lookupSongById(ctx, client, songId)
		if err != nil {
			log.Fatalf("failed to lookup song: %s", err)
		}

		fmt.Println(song.String())
	}

	fmt.Println()
	fmt.Println()

	for _, songId := range args.OtherSongIds {
		song, err := lookupSongById(ctx, client, songId)
		if err != nil {
			log.Fatalf("failed to lookup song: %s", err)
		}

		fmt.Println(song.String())
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

// How to make more useful:
// just run the tool, it parses all your memory files and:
// 1. Tells you any files that have songs that are missing fields?
// 2. Rewrites any such files with details fetched from spotify
// 3. Downloads the images in the most appropriate size, ready for renaming.

// 3pUeWeDBE5O7kttWjXFGuQ 5XTKO227Jtu81Ni41Fi9Gj 5mEyCUtI36Jmu2KNQQ4jaw 4YgqBjoGetB0h2a0s20HMY 1eZzKmzYwini2oXcgGe5zy 74HX1HcsR135apNYpHUUZj
