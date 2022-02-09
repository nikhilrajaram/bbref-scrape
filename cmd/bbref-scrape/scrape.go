package main

import (
	bbrefscrape "bbref-scrape/internal"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"golang.org/x/net/html"
)

func main() {
	season := *flag.String("season", "2022", "the season to scrape game logs for")
	ScrapeGamelogsForSeason(season)
}

func Log(format string, a ...interface{}) {
	log.Println(fmt.Sprintf(format, a...))
}

func LogFatal(a ...interface{}) {
	log.Fatal(a...)
}

func GetCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36"),
	)

	c.Limit(&colly.LimitRule{
		Parallelism: 4,
		RandomDelay: 5 * time.Second,
	})

	return c
}

// func GetGamelogFilename(name string) string {
// 	dashedName := strings.ReplaceAll(name, " ", "-")
// 	return fmt.Sprintf("output/gamelogs/%s.csv", dashedName)
// }

func ScrapeGamelogsForSeason(season string) {
	c := GetCollector()

	seasonSelector := "a[href$='" + season + ".html']"
	playerSelector := "a[href^='/players/']"

	c.OnRequest(func(r *colly.Request) {
		r.Ctx.Put("url", r.URL.String())
	})

	// visit season specified in command line args while traversing teams page
	// ex. https://www.basketball-reference.com/teams/ATL/ =>
	//     https://www.basketball-reference.com/teams/ATL/2022.html
	c.OnHTML(seasonSelector, func(e *colly.HTMLElement) {
		if e.Request.URL.String() != "https://www.basketball-reference.com/teams/" {
			return
		}
		c.Visit(e.Request.AbsoluteURL(e.Attr("href")))
	})

	// visit player gamelog page for season specified in command line args
	// while traversing team season page
	// ex. https://www.basketball-reference.com/teams/ATL/2022.html =>
	//     https://www.basketball-reference.com/players/y/youngtr01/gamelog/2022
	c.OnHTML(playerSelector, func(e *colly.HTMLElement) {
		playerUrl := e.Attr("href")
		if !strings.HasSuffix(playerUrl, ".html") ||
			!strings.Contains(playerUrl, "players") ||
			!strings.Contains(e.Request.URL.String(), season) {
			return
		}

		// gamelog pages are not of the form *.html
		playerUrl = strings.Replace(playerUrl, ".html", "", 1)
		gamelogUrl := fmt.Sprintf("%s/gamelog/%s", playerUrl, season)
		c.Visit(e.Request.AbsoluteURL(gamelogUrl))
	})

	ig := bbrefscrape.NewIdGenerator()
	im := bbrefscrape.NewIdMapper()
	// parse gamelog tables
	c.OnResponse(func(r *colly.Response) {
		url := r.Request.URL.String()
		if !strings.Contains(url, "gamelog") || !strings.Contains(url, season) {
			return
		}

		body := strings.NewReader(string(r.Body))
		doc, err := html.Parse(body)
		if err != nil {
			Log("error encountered when parsing html for gamelog %s: %+v", url, err)
			return
		}

		glTable, ok := bbrefscrape.GetGamelogTable(doc)
		if !ok {
			//todo
			Log("unable to get gamelog table ")
			return
		}

		_, stats, data := bbrefscrape.ParseGamelogTable(glTable)
		playerName, ok := bbrefscrape.GetPlayerName(doc)
		if !ok {
			Log("could not retrieve player name for gamelog %s", r.Request.URL.String())
			return
		}
		// filename := GetGamelogFilename(playerName)
		id := ig.GetId()
		filename := fmt.Sprintf("output/gamelogs/%d.csv", id)
		im.SetName(id, playerName)

		f, err := os.Create(filename)
		if err != nil {
			LogFatal(err)
		}
		defer f.Close()

		csv := csv.NewWriter(f)

		write := func(data []string) {
			defer csv.Flush()
			err = csv.Write(data)
			if err != nil {
				LogFatal(err)
			}
		}

		write(stats)
		for i := range data {
			write(data[i])
		}
	})

	c.Visit("https://www.basketball-reference.com/teams/")
	c.Wait()

	im.Dump("output/gamelogs/index.json")
}
