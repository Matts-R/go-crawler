package main

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anthdm/hollywood/actor"
	"golang.org/x/net/html"
)

type VisitRequest struct {
	links []string
}

type Visitor struct {
	managerPID *actor.PID
	URL        string
}

func NewVisitor(url string, mpid *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &Visitor{
			URL:        url,
			managerPID: mpid,
		}
	}
}

func (v *Visitor) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		slog.Info("visitor has started", "url", v.URL)
		links, err := doVisit(v.URL)

		if err != nil {
			slog.Error("visit error", "err", err)
			return
		}

		c.Send(v.managerPID, VisitRequest{links})
	case actor.Stopped:

	}
}

type Manager struct {
	visitors map[*actor.PID]bool
}

func NewManager() actor.Producer {
	return func() actor.Receiver {
		return &Manager{
			visitors: make(map[*actor.PID]bool),
		}
	}
}

func (m *Manager) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case VisitRequest:
		m.handleVisitRequest(c, msg)
	case actor.Started:
		slog.Info("manager has started")
	case actor.Stopped:

	}
}

func (m *Manager) handleVisitRequest(c *actor.Context, msg VisitRequest) error {
	for _, link := range msg.links {
		slog.Info("visiting url", "url", link)
		c.SpawnChild(NewVisitor(link, c.PID()), "visitor/"+link)
	}

	return nil
}

func main() {
	e, err := actor.NewEngine(actor.NewEngineConfig())

	if err != nil {
		log.Fatal(err)
	}

	pid := e.Spawn(NewManager(), "manager")

	e.Send(pid, VisitRequest{links: []string{"https://levenue.com"}})
	e.Send(pid, VisitRequest{links: []string{"https://fulltimegodev.com"}})

	time.Sleep(time.Second * 10)
}

func extractLinks(body io.Reader) ([]string, error) {
	links := make([]string, 0)
	tokenizer := html.NewTokenizer(body)

	for {
		tokenType := tokenizer.Next()

		if tokenType == html.ErrorToken {
			return links, nil
		}

		if tokenType == html.StartTagToken {
			token := tokenizer.Token()

			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
	}
}

func removeDuplicateLinks(links []string) []string {
	seen := make(map[string]bool)
	unique := make([]string, 0)

	for _, link := range links {
		if !seen[link] {
			unique = append(unique, link)
			seen[link] = true
		}
	}

	return unique
}

func doVisit(baseLink string) ([]string, error) {
	baseUrl, err := url.Parse(baseLink)

	if err != nil {
		return []string{}, err
	}

	resp, err := http.Get(baseUrl.String())

	if err != nil {
		return []string{}, err
	}

	links, err := extractLinks(resp.Body)

	if err != nil {
		return []string{}, err
	}

	for i, link := range links {
		if !strings.Contains(link, "https") {
			links[i] = baseLink + link
		}
	}

	links = removeDuplicateLinks(links)
	return links, nil

	// for _, link := range links {
	// 	linkUrl, err := url.Parse(link)

	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	_ = baseUrl.ResolveReference(linkUrl)

	// 	linksList = append(linksList, baseUrl.ResolveReference(linkUrl))
	// }
}
