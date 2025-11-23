package sonostalgia

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
)

// All the parsed template params
type Sonostalgia struct {
	IndexParams    Index
	AboutParams    About
	MemoriesParams Memories
	YearsParams    Years
	MemoryParams   []Memory
}

func LoadSonostalgia(memoryFiles []string) (*Sonostalgia, error) {
	var memories []Memory
	for _, file := range memoryFiles {
		memory, err := LoadMemory(file)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", file, err)
		}
		memories = append(memories, *memory)
	}

	// Index
	//MemoryCount = just take len
	// Hmm unique songs?? uniqueuness is artist x title
	//SongCount = sum(len(songs)) over all memories
	//ArtistCount = sum(len(songs)) over all memories
	//YearsWithEntries = unique 2xxx numbers that appear in dates?
	// this will undercount it I do ranges like 2017-2019
	//RecentMemories = ??? take some from the current year?

	var (
		memoryCount        = len(memories)
		songCount          int
		artistCount        int
		yearsWithEntries   int
		earliestMemoryYear string
		recentMemories     []Memory
		yearsForParams     []Year
	)

	// string is of form "song|artist"
	songSet := map[string]struct{}{}
	artistSet := map[string]struct{}{}
	yearSet := map[string][]Memory{}

	for _, memory := range memories {

		// Add to song and artist sets.
		// Account for duplicate songs, and songs with same name
		for _, song := range memory.Songs {
			songSet[fmt.Sprintf("%s|%s", song.Name, song.Artist)] = struct{}{}
			artistSet[song.Artist] = struct{}{}
		}

		parsedDates := parseDateString(memory.Date)
		for _, year := range parsedDates {
			if yearMemories, ok := yearSet[year]; ok {
				yearSet[year] = append(yearMemories, memory)
			} else {
				yearSet[year] = []Memory{memory}
			}
		}
	}

	yearsForParams = []Year{}

	minYear := 3000
	for year, yearMemories := range yearSet {
		yearInt, err := strconv.Atoi(year)
		if err != nil {
			continue
		}
		if yearInt < minYear {
			minYear = yearInt
		}
		yearsForParams = append(yearsForParams, Year{
			Year:     yearInt,
			Memories: yearMemories,
		})
	}

	sort.Slice(yearsForParams, func(i, j int) bool {
		return yearsForParams[i].Year < yearsForParams[j].Year
	})

	songCount = len(songSet)
	artistCount = len(artistSet)
	yearsWithEntries = len(yearSet)
	recentMemories = memories[max(len(memories), 5)-5:] // TODO
	earliestMemoryYear = strconv.Itoa(minYear)

	return &Sonostalgia{
		AboutParams: About{
			EarliestMemory: earliestMemoryYear,
			MemoryCount:    memoryCount,
			SongCount:      songCount,
			ArtistCount:    artistCount,
		},
		IndexParams: Index{
			MemoryCount:      memoryCount,
			SongCount:        songCount,
			ArtistCount:      artistCount,
			YearsWithEntries: yearsWithEntries,
			RecentMemories:   recentMemories,
		},
		MemoryParams: memories,
		MemoriesParams: Memories{
			AllMemories: memories,
		},
		YearsParams: Years{
			Years: yearsForParams,
		},
	}, nil
}

// it's freeform but we'll hope for the following:
// A single year: "2019"
// A range of years: "2019-2022"
// A list of years: 2019,2020,2022
func parseDateString(dateString string) []string {
	dateString = strings.TrimSpace(dateString)

	if strings.Contains(dateString, "-") {
		log.Printf("trying to parse date %s as a range\n", dateString)
		rangeDates := strings.Split(dateString, "-")
		if len(rangeDates) != 2 {
			log.Printf("thought date %s was a range but had too wrong number of parts\n", dateString)
			return []string{}
		}
		// parse to ints, range between the two, convert back to stirng and add to set
		begin, err := strconv.Atoi(rangeDates[0])
		if err != nil {
			log.Printf("failed to convert start year to int\n")
			return []string{}
		}
		end, err := strconv.Atoi(rangeDates[1])
		if err != nil {
			log.Printf("failed to convert start year to int\n")
			return []string{}
		}
		dates := []string{}
		for i := begin; i <= end; i++ {
			dates = append(dates, strconv.Itoa(i))
		}
		log.Printf("dates: %s\n", dates)
		return dates
	} else if strings.Contains(dateString, ",") {
		log.Printf("trying to parse date %s as a list\n", dateString)
		dates := strings.Split(dateString, ",")
		log.Printf("dates: %s\n", dates)
		return dates
	} else {
		// assume it's fine
		log.Printf("trying to parse date %s as a single year\n", dateString)
		dates := []string{dateString}
		log.Printf("dates: %s\n", dates)
		return dates
	}
}
