package javbus

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/metatube-community/metatube-sdk-go/common/fetch"
	"github.com/metatube-community/metatube-sdk-go/common/parser"
	"github.com/metatube-community/metatube-sdk-go/model"
	"github.com/metatube-community/metatube-sdk-go/provider"
	"github.com/metatube-community/metatube-sdk-go/provider/internal/scraper"
	"github.com/PuerkitoBio/goquery"
)

var _ provider.MovieProvider = (*JavBus)(nil)

const (
	Name     = "JavBus"
	Priority = 1000
)

const (
	baseURL = "https://www.javbus.com"
)

type JavBus struct {
	*scraper.Scraper
}

func New() *JavBus {
	return &JavBus{
		Scraper: scraper.NewDefaultScraper(Name, baseURL, Priority,
			scraper.WithFetchOption(fetch.WithHTTPClient(&http.Client{
				Transport: fetch.DefaultTransport(),
			})),
			scraper.WithFetchOption(fetch.WithHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")),
		),
	}
}

func (bus *JavBus) NormalizeMovieKeyword(keyword string) string {
	return strings.ToUpper(keyword)
}

func (bus *JavBus) NormalizeMovieID(id string) string {
	return strings.ToUpper(id)
}

func (bus *JavBus) SearchMovie(ctx context.Context, keyword string) (results []*model.MovieSearchResult, err error) {
	query := url.Values{}
	query.Set("search", keyword)
	searchURL := bus.BaseURL + "/search?" + query.Encode()

	doc, err := bus.FetchHTML(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	doc.Find("div#waterfall div.item").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Find("a").Attr("href")
		if !exists {
			return
		}
		movieID, err := bus.ParseMovieIDFromURL(href)
		if err != nil {
			return
		}
		title := strings.TrimSpace(s.Find("a").Text())
		imgSrc, _ := s.Find("img").Attr("src")
		if !strings.HasPrefix(imgSrc, "http") {
			imgSrc = baseURL + imgSrc
		}

		results = append(results, &model.MovieSearchResult{
			Provider: bus.Name(),
			ID:       movieID,
			Title:    title,
			CoverURL: imgSrc,
			URL:      href,
		})
	})
	return
}

func (bus *JavBus) GetMovieInfoByID(ctx context.Context, id string) (info *model.MovieInfo, err error) {
	id = bus.NormalizeMovieID(id)
	return bus.GetMovieInfoByURL(ctx, fmt.Sprintf("%s/%s", bus.BaseURL, id))
}

func (bus *JavBus) GetMovieInfoByURL(ctx context.Context, rawURL string) (info *model.MovieInfo, err error) {
	doc, err := bus.FetchHTML(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	info = &model.MovieInfo{
		Provider: bus.Name(),
		URL:      rawURL,
	}
	info.ID, err = bus.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse id failed: %w", err)
	}
	info.Title = strings.TrimSpace(doc.Find("h3").Text())

	// 解析发行日期
	info.ReleaseDate = parser.ParseDate(doc.Find("div.info p:contains(发行日期)").Text())

	// 解析演员
	doc.Find("div.info p:contains(演员) a").Each(func(_ int, sel *goquery.Selection) {
		info.Actors = append(info.Actors, strings.TrimSpace(sel.Text()))
	})

	// 封面
	coverSrc, ok := doc.Find("div.bigImage img").Attr("src")
	if ok {
		if !strings.HasPrefix(coverSrc, "http") {
			coverSrc = baseURL + coverSrc
		}
		info.CoverURL = coverSrc
	}

	// 解析番号、系列、厂商
	doc.Find("div.info p").Each(func(_ int, sel *goquery.Selection) {
		txt := strings.TrimSpace(sel.Text())
		switch {
		case strings.Contains(txt, "番号:"):
			info.Code = strings.TrimSpace(strings.TrimPrefix(txt, "番号:"))
		case strings.Contains(txt, "发行商:"):
			info.Studio = strings.TrimSpace(strings.TrimPrefix(txt, "发行商:"))
		case strings.Contains(txt, "系列:"):
			info.Series = strings.TrimSpace(strings.TrimPrefix(txt, "系列:"))
		}
	})
	return info, nil
}

func (bus *JavBus) ParseMovieIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	segments := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	if len(segments) == 0 {
		return "", fmt.Errorf("no id found in url")
	}
	return segments[len(segments)-1], nil
}
