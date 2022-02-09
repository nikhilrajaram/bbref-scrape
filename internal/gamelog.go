package bbrefscrape

import (
	"regexp"
	"strconv"

	"golang.org/x/net/html"
)

// dfs search for div[id='all_pgl_basic'] element
func GetGamelogDiv(n *html.Node) (*html.Node, bool) {
	return Search(n, func(n *html.Node) bool {
		if n.Data == "div" {
			id, ok := GetAttribute(n, "id")
			return ok && id == "all_pgl_basic"
		}
		return false
	})
}

// dfs search for table element
// wrt bbref, presumes input is gamelog div node retrieved from GetGamelogDiv
func GetGamelogTableSearch(n *html.Node) (*html.Node, bool) {
	if n.Type == html.ElementNode && n.Data == "table" {
		return n, true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		table, ok := GetGamelogTableSearch(c)
		if ok {
			return table, true
		}
	}

	return nil, false
}

// gets gamelog table when passed parsed gamelog page dom as input
func GetGamelogTable(n *html.Node) (*html.Node, bool) {
	div, ok := GetGamelogDiv(n)
	if !ok {
		return nil, false
	}

	table, ok := GetGamelogTableSearch(div)
	if !ok {
		return nil, false
	}

	return table, true
}

// parses table headers
// presumes input is thead of gamelog table
// returns labels (from aria-label attribute) and stat abbreviations (retrieved
//   from data-stat attribute)
func ParseTableHeaders(thead *html.Node) ([]string, []string) {
	labels := make([]string, 0)
	stats := make([]string, 0)
	for c := thead.FirstChild; c != nil; c = c.NextSibling {
		if c.Data != "tr" {
			continue
		}

		for th := c.FirstChild; th != nil; th = th.NextSibling {
			if th.Data != "th" {
				continue
			}

			label, ok := GetAttribute(th, "aria-label")
			if !ok {
				Log("skipping table header for which aria-label attribute not present: %+v", th)
				continue
			}

			stat, ok := GetAttribute(th, "data-stat")
			if !ok {
				Log("skipping table header for which data-stat attribute not present: %+v", th)
				continue
			}

			labels = append(labels, label)
			stats = append(stats, stat)
		}
	}

	return labels, stats
}

// returns the data in a 2d table given a tbody node
func ParseTableData(tbody *html.Node, stats []string) [][]string {
	data := make([][]string, 0)

	// below is simply used for fast lookups of data-stat attributes
	statSet := make(map[string]int, 0)
	for i := range stats {
		statSet[stats[i]] = i
	}

	// iterate through rows of table
	for c := tbody.FirstChild; c != nil; c = c.NextSibling {
		if c.Data != "tr" {
			continue
		}

		class, ok := GetAttribute(c, "class")
		if ok {
			// skip the repeated header rows in the gamelog table
			if class == "thead" {
				continue
			}
		}

		datum := make([]string, 0)
		for td := c.FirstChild; td != nil; td = td.NextSibling {
			if td.Data != "th" && td.Data != "td" {
				continue
			}
			stat, ok := GetAttribute(td, "data-stat")

			if !ok {
				Log("data-stat attribute not present in table cell %+v", td)
				continue
			}

			_, ok = statSet[stat]
			if !ok {
				Log("encountered stat not included in headers: %s. filling remaining cells with null values", stat)

				text, _ := GetText(td)
				colSpan, ok := GetAttribute(td, "colspan")
				if !ok {
					Log("colspan not present for table cell %+v. adding single null cell", td)
					datum = append(datum, text)
					continue
				}

				n, err := strconv.Atoi(colSpan)
				if err != nil {
					Log("error encountered when converting table cell %+v colspan attribute to integer: %+v", td, err)
					continue
				}

				for j := 0; j < n; j++ {
					datum = append(datum, text)
				}
				continue
			}

			text, ok := GetText(td)
			if !ok {
				Log("text not present for table cell %+v", td)
				datum = append(datum, "")
				continue
			}
			datum = append(datum, text)
		}
		data = append(data, datum)
	}
	return data
}

// parses labels (headers), data-stat attributes, from the gamelog table on a
// gamelog page
func ParseGamelogTable(n *html.Node) ([]string, []string, [][]string) {
	var labels, stats []string
	var data [][]string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Data {
		case "thead":
			labels, stats = ParseTableHeaders(c)
		case "tbody":
			data = ParseTableData(c, stats)
		default:
			continue
		}
	}

	return labels, stats, data
}

// gets player name from a gamelog page
func GetPlayerName(n *html.Node) (string, bool) {
	if n.Data == "h1" {
		itemProp, ok := GetAttribute(n, "itemprop")
		if !ok || itemProp != "name" {
			return "", false
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Data != "span" {
				continue
			}

			text, ok := GetText(c)
			if !ok || len(text) == 0 {
				return "", false
			}

			r := regexp.MustCompile(` [0-9][0-9][0-9][0-9]-[0-9][0-9]`)
			i := r.FindStringIndex(text)[0]
			return text[:i], true
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		name, ok := GetPlayerName(c)
		if ok {
			return name, true
		}
	}

	return "", false
}
