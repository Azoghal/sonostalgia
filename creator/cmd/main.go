package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	sonostalgia "github.com/azoghal/sonostalgia/src"
	"github.com/joho/godotenv"
	spotify "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
	"gopkg.in/yaml.v3"
)

//go:embed index.html
var indexHTML []byte

const (
	minDesiredWidth = 100
	maxDesiredWidth = 350
)

var (
	spotifyURLRe  = regexp.MustCompile(`open\.spotify\.com/track/([A-Za-z0-9]+)`)
	spotifyURIRe  = regexp.MustCompile(`^spotify:track:([A-Za-z0-9]+)$`)
	validSlugRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
)

type server struct {
	client *spotify.Client
	ctx    context.Context
}

type SearchRequest struct {
	Query string `json:"query"`
}

// SongResult is returned by both /api/search and /api/fetch-song.
type SongResult struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	SongLink  string               `json:"songLink"`
	Artists   []sonostalgia.Artist `json:"artists"`
	AlbumName string               `json:"albumName"`
	ImageURL  string               `json:"imageUrl"`
	ImageName string               `json:"imageName"`
}

type FetchRequest struct {
	URL string `json:"url"`
}

type SaveSong struct {
	Name            string               `json:"name"`
	SongLink        string               `json:"songLink"`
	Artists         []sonostalgia.Artist `json:"artists"`
	RelevantDate    string               `json:"relevantDate"`
	ImageName       string               `json:"imageName"`
	SpotifyImageURL string               `json:"spotifyImageUrl"`
}

type SaveRequest struct {
	OutputTitle string     `json:"outputTitle"`
	Title       string     `json:"title"`
	ShortTitle  string     `json:"shortTitle"`
	Subtitle    string     `json:"subtitle"`
	Date        string     `json:"date"`
	Content     string     `json:"content"`
	Songs       []SaveSong `json:"songs"`
	OtherSongs  []SaveSong `json:"otherSongs"`
	Rebuild     bool       `json:"rebuild"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("failed to load .env")
	}

	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		log.Fatalf("couldn't get spotify token: %v", err)
	}

	s := &server{
		client: spotify.New(spotifyauth.New().Client(ctx, token)),
		ctx:    ctx,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	http.HandleFunc("/api/search", s.handleSearch)
	http.HandleFunc("/api/fetch-song", s.handleFetchSong)
	http.HandleFunc("/api/save", s.handleSave)

	addr := ":8765"
	fmt.Printf("Sonostalgia Creator → http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	res, err := s.client.Search(s.ctx, req.Query, spotify.SearchTypeTrack, spotify.Limit(8), spotify.Market("GB"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]SongResult, 0, len(res.Tracks.Tracks))
	for _, t := range res.Tracks.Tracks {
		artists := make([]sonostalgia.Artist, len(t.Artists))
		for i, a := range t.Artists {
			artists[i] = sonostalgia.Artist{Name: a.Name, Link: a.ExternalURLs["spotify"]}
		}
		out = append(out, SongResult{
			ID:        t.ID.String(),
			Name:      t.Name,
			SongLink:  t.ExternalURLs["spotify"],
			Artists:   artists,
			AlbumName: t.Album.Name,
			ImageURL:  bestImageURL(t.Album.Images),
			ImageName: makeImageName(t.Name),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *server) handleFetchSong(w http.ResponseWriter, r *http.Request) {
	var req FetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	id := extractTrackID(req.URL)
	track, err := s.client.GetTrack(s.ctx, spotify.ID(id), spotify.Market("GB"))
	if err != nil {
		http.Error(w, fmt.Sprintf("track lookup failed: %v", err), http.StatusBadGateway)
		return
	}

	album, err := s.client.GetAlbum(s.ctx, track.Album.ID, spotify.Market("GB"))
	if err != nil {
		http.Error(w, fmt.Sprintf("album lookup failed: %v", err), http.StatusBadGateway)
		return
	}

	artists := make([]sonostalgia.Artist, len(track.Artists))
	for i, a := range track.Artists {
		artists[i] = sonostalgia.Artist{Name: a.Name, Link: a.ExternalURLs["spotify"]}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SongResult{
		ID:        track.ID.String(),
		Name:      track.Name,
		SongLink:  track.ExternalURLs["spotify"],
		Artists:   artists,
		AlbumName: album.Name,
		ImageURL:  bestImageURL(album.Images),
		ImageName: makeImageName(track.Name),
	})
}

func (s *server) handleSave(w http.ResponseWriter, r *http.Request) {
	var req SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !validSlugRe.MatchString(req.OutputTitle) {
		http.Error(w, "outputTitle must be lowercase alphanumeric with hyphens", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll("src/assets", 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	songs, err := processSongs(req.Songs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	otherSongs, err := processSongs(req.OtherSongs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mem := sonostalgia.Memory{
		OutputTitle: req.OutputTitle,
		PageTitle:   req.ShortTitle,
		Title:       req.Title,
		Subtitle:    req.Subtitle,
		Date:        req.Date,
		Content:     req.Content,
		Songs:       songs,
		OtherSongs:  otherSongs,
	}

	data, err := yaml.Marshal(mem)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	yamlPath := fmt.Sprintf("src/memories/%s.yaml", req.OutputTitle)
	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("saved %s", yamlPath)

	if req.Rebuild {
		out, err := exec.Command("./build/templater").CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("rebuild failed: %v\n%s", err, out), http.StatusInternalServerError)
			return
		}
		cmd := exec.Command("sh", "-c", "mkdir -p ./output/assets && cp -r ./src/assets/. ./output/assets/")
		if err := cmd.Run(); err != nil {
			log.Printf("warning: asset copy failed: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": yamlPath})
}

func processSongs(songs []SaveSong) ([]sonostalgia.Song, error) {
	out := make([]sonostalgia.Song, 0, len(songs))
	for _, s := range songs {
		imageLink := ""
		if s.SpotifyImageURL != "" && s.ImageName != "" {
			dest := fmt.Sprintf("src/assets/%s.jpg", s.ImageName)
			if err := downloadImage(s.SpotifyImageURL, dest); err != nil {
				log.Printf("warning: failed to download image for %q: %v", s.Name, err)
			} else {
				imageLink = fmt.Sprintf("assets/%s.jpg", s.ImageName)
			}
		}
		out = append(out, sonostalgia.Song{
			Name:         s.Name,
			SongLink:     s.SongLink,
			Artists:      s.Artists,
			RelevantDate: s.RelevantDate,
			ImageLink:    imageLink,
		})
	}
	return out, nil
}

func extractTrackID(s string) string {
	if m := spotifyURLRe.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	if m := spotifyURIRe.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return strings.TrimSpace(s)
}

func bestImageURL(images []spotify.Image) string {
	var best *spotify.Image
	bestScore := 0
	for i, img := range images {
		w := int(img.Width)
		score := 1
		if w < maxDesiredWidth {
			score++
		}
		if w > minDesiredWidth {
			score++
		}
		if score > bestScore {
			best = &images[i]
			bestScore = score
		}
	}
	if best == nil {
		return ""
	}
	return best.URL
}

func makeImageName(trackName string) string {
	alpha := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, trackName)
	return strings.Join(strings.Fields(strings.ToLower(alpha)), "-")
}

func downloadImage(url, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
