package main

import (
	"context"
	"errors"
	"log"
	"os"

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

func main() {
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

	testId := "52ug4BzkXF06EkVX8axf8y"

	song, err := lookupSongById(ctx, client, testId)
	if err != nil {
		log.Fatalf("failed to lookup song: %s", err)
	}

	log.Println(song.String())
}

func lookupSongById(ctx context.Context, client *spotify.Client, id string) (*sonostalgia.Song, error) {

	testId := spotify.ID(id)

	track, err := client.GetTrack(ctx, testId, spotify.Market("GB"))
	if err != nil {
		return nil, errors.New("track request failed")
	}

	log.Printf("TRACK: %s\nARTISTS: %s\nALBUM_ID: %s\n", track.Name, track.Artists, track.Album.ID)

	album, err := client.GetAlbum(ctx, track.Album.ID, spotify.Market("GB"))
	if err != nil {
		return nil, errors.New("album request failed")
	}

	song := &sonostalgia.Song{
		Name:       track.Name,
		Artist:     track.Artists[0].Name, // TODO need to handle multiple artists
		SongLink:   track.ExternalURLs["spotify"],
		ArtistLink: track.Artists[0].ExternalURLs["spotify"],
		ImageLink:  album.Images[len(album.Images)-1].URL, // TODO do something more sensible here to get the right size
	}

	return song, nil
}
