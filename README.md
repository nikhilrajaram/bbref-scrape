# bbref-scrape
Program to scrape https://www.basketball-reference.com/

## Currently supporting
 * gamelogs for all players within a season

## How to use
Compile the program via
```bash
go build cmd/bbref-scrape/scrape.go
```

And execute it with
```bash
./scrape <season_year_end>
```

Once executed, the program will dump the scraped gamelogs to `output/gamelogs/`.