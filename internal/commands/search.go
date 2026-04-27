package commands

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/ui"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

func searchCommand() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "search for packages across all available backends",
		ArgsUsage: "<query>",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() == 0 {
				return errors.New("query required")
			}
			query := c.Args().First()

			ac, err := loadContext(c)
			if err != nil {
				return err
			}

			var (
				mu     sync.Mutex
				wg     sync.WaitGroup
				groups []ui.SearchResultGroup
			)

			for _, b := range ac.registry.All() {
				s, ok := b.(backend.Searcher)
				if !ok || !b.Available() {
					continue
				}
				wg.Add(1)
				go func(name string, s backend.Searcher) {
					defer wg.Done()
					results, err := s.Search(ctx, query)
					mu.Lock()
					groups = append(
						groups,
						ui.SearchResultGroup{Name: name, Results: results, Err: err},
					)
					mu.Unlock()
				}(b.Name(), s)
			}
			wg.Wait()

			if len(groups) == 0 {
				ui.Warn("no backends support search")
				return nil
			}

			total := 0
			for _, g := range groups {
				if g.Err == nil {
					total += len(g.Results)
				}
			}
			if total == 0 {
				ui.Info(fmt.Sprintf("no results for %q", query))
				return nil
			}

			ui.PrintSearchResults(groups, query)
			return nil
		},
	}
}
