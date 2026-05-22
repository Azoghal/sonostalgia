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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
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

//go:embed login.html
var loginHTML []byte

const (
	minDesiredWidth = 100
	maxDesiredWidth = 350
	wipsPath        = "src/wip-memories/ideas.yaml"
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
	Name              string               `json:"name"`
	SongLink          string               `json:"songLink"`
	Artists           []sonostalgia.Artist `json:"artists"`
	RelevantDate      string               `json:"relevantDate"`
	ImageName         string               `json:"imageName"`
	SpotifyImageURL   string               `json:"spotifyImageUrl"`
	ExistingImageLink string               `json:"existingImageLink"`
}

type MemoryListItem struct {
	OutputTitle string `json:"outputTitle"`
	Title       string `json:"title"`
}

// MemoryResponse is used for /api/memory — gives the frontend predictable camelCase keys.
type MemoryResponse struct {
	OutputTitle string         `json:"outputTitle"`
	ShortTitle  string         `json:"shortTitle"`
	Title       string         `json:"title"`
	Subtitle    string         `json:"subtitle"`
	Date        string         `json:"date"`
	Content     string         `json:"content"`
	Songs       []SongResponse `json:"songs"`
	OtherSongs  []SongResponse `json:"otherSongs"`
}

type SongResponse struct {
	Name         string           `json:"name"`
	SongLink     string           `json:"songLink"`
	Artists      []ArtistResponse `json:"artists"`
	RelevantDate string           `json:"relevantDate"`
	ImageLink    string           `json:"imageLink"`
}

type ArtistResponse struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

type WIPEntry struct {
	ID      string `yaml:"id"      json:"id"`
	Title   string `yaml:"title"   json:"title"`
	Notes   string `yaml:"notes"   json:"notes"`
	Created string `yaml:"created" json:"created"`
}

type AddWIPRequest struct {
	Title string `json:"title"`
	Notes string `json:"notes"`
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

	secret := os.Getenv("AUTH_SECRET")
	if secret == "" {
		log.Fatal("AUTH_SECRET must be set in .env")
	}

	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		TokenURL:     spotifyauth.TokenURL,
	}

	s := &server{
		client: spotify.New(config.Client(ctx)),
		ctx:    ctx,
	}

	// Authenticated routes — all behind the cookie check.
	authed := http.NewServeMux()
	authed.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	authed.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("src/assets"))))
	authed.HandleFunc("/api/memories", s.handleListMemories)
	authed.HandleFunc("/api/memory", s.handleGetMemory)
	authed.HandleFunc("/api/search", s.handleSearch)
	authed.HandleFunc("/api/fetch-song", s.handleFetchSong)
	authed.HandleFunc("/api/save", s.handleSave)
	authed.HandleFunc("/api/wips", s.handleWIPs)
	authed.HandleFunc("/api/wip", s.handleDeleteWIP)

	// Top-level mux: login routes are public, everything else is protected.
	mux := http.NewServeMux()
	mux.HandleFunc("/login", handleLoginPage(loginHTML))
	mux.HandleFunc("/api/login", makeLoginHandler(secret))
	mux.Handle("/", authMiddleware(secret, authed))

	addr := ":8765"
	fmt.Printf("Sonostalgia Creator → http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func loadWIPs() ([]WIPEntry, error) {
	data, err := os.ReadFile(wipsPath)
	if os.IsNotExist(err) {
		return []WIPEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []WIPEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []WIPEntry{}
	}
	return entries, nil
}

func saveWIPs(entries []WIPEntry) error {
	if err := os.MkdirAll("src/wip-memories", 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(wipsPath, data, 0644)
}

func (s *server) handleWIPs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		entries, err := loadWIPs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)

	case http.MethodPost:
		var req AddWIPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}
		entries, err := loadWIPs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entry := WIPEntry{
			ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
			Title:   strings.TrimSpace(req.Title),
			Notes:   strings.TrimSpace(req.Notes),
			Created: time.Now().Format("2006-01-02"),
		}
		entries = append(entries, entry)
		if err := saveWIPs(entries); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleDeleteWIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	entries, err := loadWIPs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filtered := entries[:0]
	for _, e := range entries {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	if err := saveWIPs(filtered); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("src/memories/*.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]MemoryListItem, 0, len(files))
	for _, f := range files {
		mem, err := sonostalgia.LoadMemory(f)
		if err != nil {
			log.Printf("warning: skipping %s: %v", f, err)
			continue
		}
		items = append(items, MemoryListItem{OutputTitle: mem.OutputTitle, Title: mem.Title})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *server) handleGetMemory(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if !validSlugRe.MatchString(slug) {
		http.Error(w, "invalid slug", http.StatusBadRequest)
		return
	}

	mem, err := sonostalgia.LoadMemory(fmt.Sprintf("src/memories/%s.yaml", slug))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memoryToResponse(mem))
}

func memoryToResponse(mem *sonostalgia.Memory) MemoryResponse {
	mapSongs := func(songs []sonostalgia.Song) []SongResponse {
		out := make([]SongResponse, len(songs))
		for i, s := range songs {
			artists := make([]ArtistResponse, len(s.Artists))
			for j, a := range s.Artists {
				artists[j] = ArtistResponse{Name: a.Name, Link: a.Link}
			}
			out[i] = SongResponse{
				Name:         s.Name,
				SongLink:     s.SongLink,
				Artists:      artists,
				RelevantDate: s.RelevantDate,
				ImageLink:    s.ImageLink,
			}
		}
		return out
	}

	return MemoryResponse{
		OutputTitle: mem.OutputTitle,
		ShortTitle:  mem.PageTitle,
		Title:       mem.Title,
		Subtitle:    mem.Subtitle,
		Date:        mem.Date,
		Content:     mem.Content,
		Songs:       mapSongs(mem.Songs),
		OtherSongs:  mapSongs(mem.OtherSongs),
	}
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
		imageLink := s.ExistingImageLink
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
