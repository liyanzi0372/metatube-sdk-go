package javbus

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/metatube-community/metatube-sdk-go/model"
	"github.com/metatube-community/metatube-sdk-go/provider"
	"github.com/metatube-community/metatube-sdk-go/scraper"
)

var _ provider.MovieProvider = (*JavBus)(nil)

const (
	Name     = "JavBus"
	Priority = 1000
	baseURL  = "https://www.javbus.com"
	ua       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

type JavBus struct {
	sc *scraper.Scraper
}

func New() *JavBus {
	return &JavBus{
		sc: scraper.New(),
	}
}

func (bus *JavBus) Name() string {
	return Name
}

func (bus *JavBus) Priority() int {
	return Priority
}

// 【新版接口：不带context】
func (bus *JavBus) GetMovieInfoByID(id string) (*model.MovieInfo, error) {
	link := fmt.Sprintf("%s/%s", baseURL, id)
	html, err := bus.sc.FetchHTML(link, scraper.WithHeader("User-Agent", ua))
	if err != nil {
		return nil, err
	}

	info := &model.MovieInfo{
		Provider: Name,
		ID:       id,
		URL:      link,
	}

	// 标题
	titleRe := regexp.MustCompile(`<h3>([^<]+)</h3>`)
	titleMatch := titleRe.FindStringSubmatch(html)
	if len(titleMatch) > 1 {
		info.Title = strings.TrimSpace(titleMatch[1])
	}

	// 封面图
	imgRe := regexp.MustCompile(`bigImage" href="([^"]+)"`)
	imgMatch := imgRe.FindStringSubmatch(html)
	if len(imgMatch) > 1 {
		info.CoverURL = imgMatch[1]
	}

	// 发行日期
	dateRe := regexp.MustCompile(`<td>发行日期:</td>\s*<td>(\d{4}-\d{2}-\d{2})</td>`)
	dateMatch := dateRe.FindStringSubmatch(html)
	if len(dateMatch) > 1 {
		info.ReleaseDate = dateMatch[1]
	}

	return info, nil
}

// 【新版接口：不带context】
func (bus *JavBus) SearchMovie(keyword string) ([]*model.MovieSearchResult, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("search", keyword)
	u.RawQuery = q.Encode()
	targetURL := u.String()

	html, err := bus.sc.FetchHTML(targetURL, scraper.WithHeader("User-Agent", ua))
	if err != nil {
		return nil, err
	}

	var list []*model.MovieSearchResult
	re := regexp.MustCompile(`<a class="movie-box" href="([^"]+)">[\s\S]*?<img src="([^"]+)"[\s\S]*?<div class="photo-info">([\s\S]+?)</div>`)
	all := re.FindAllStringSubmatch(html, -1)
	for _, item := range all {
		movieURL := item[1]
		movieID := extractID(movieURL)
		titleLine := strings.TrimSpace(item[3])
		list = append(list, &model.MovieSearchResult{
			Provider: Name,
			ID:       movieID,
			Title:    titleLine,
			URL:      baseURL + movieURL,
			CoverURL: item[2],
		})
	}
	return list, nil
}

func extractID(raw string) string {
	raw = strings.Trim(raw, "/")
	parts := strings.Split(raw, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return raw
}
