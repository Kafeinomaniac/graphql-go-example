package loader

import (
	"context"
	"sync"

	"github.com/tonyghita/graphql-go-example/errors"
	"github.com/tonyghita/graphql-go-example/swapi"

	"github.com/nicksrandall/dataloader"
)

func LoadStarship(ctx context.Context, url string) (swapi.Starship, error) {
	var ship swapi.Starship

	ldr, err := extract(ctx, starshipLoaderKey)
	if err != nil {
		return ship, err
	}

	data, err := ldr.Load(ctx, url)()
	if err != nil {
		return ship, err
	}

	ship, ok := data.(swapi.Starship)
	if !ok {
		return ship, errors.UnexpectedResponse
	}

	return ship, nil
}

func LoadStarships(ctx context.Context, urls []string) (StarshipResults, error) {
	var results []StarshipResult

	ldr, err := extract(ctx, starshipLoaderKey)
	if err != nil {
		return results, err
	}

	data, errs := ldr.LoadMany(ctx, urls)()
	results = make([]StarshipResult, 0, len(urls))

	for i, d := range data {
		var e error
		if errs != nil {
			e = errs[i]
		}

		ship, ok := d.(swapi.Starship)
		if !ok && e == nil {
			e = errors.UnexpectedResponse
		}

		results = append(results, StarshipResult{Starship: ship, Error: e})
	}

	return results, nil
}

type StarshipResult struct {
	Starship swapi.Starship
	Error    error
}

type StarshipResults []StarshipResult

func (results StarshipResults) WithoutErrors() []swapi.Starship {
	ships := make([]swapi.Starship, 0, len(results))

	for _, r := range results {
		if r.Error != nil {
			continue
		}
		ships = append(ships, r.Starship)
	}

	return ships
}

type starshipGetter interface {
	Starship(ctx context.Context, url string) (swapi.Starship, error)
}

type StarshipLoader struct {
	get starshipGetter
}

func newStarshipLoader(client starshipGetter) dataloader.BatchFunc {
	return StarshipLoader{get: client}.loadBatch
}

func PrimeStarships(ctx context.Context, page swapi.StarshipPage) error {
	ldr, err := extract(ctx, starshipLoaderKey)
	if err != nil {
		return err
	}

	for _, s := range page.Starships {
		ldr.Prime(s.URL, s)
	}

	return nil
}

func (ldr StarshipLoader) loadBatch(ctx context.Context, urls []string) []*dataloader.Result {
	var (
		n       = len(urls)
		results = make([]*dataloader.Result, n)
		wg      sync.WaitGroup
	)

	wg.Add(n)

	for i, url := range urls {
		go func(ctx context.Context, url string, i int) {
			data, err := ldr.get.Starship(ctx, url)
			results[i] = &dataloader.Result{Data: data, Error: err}
			wg.Done()
		}(ctx, url, i)
	}

	wg.Wait()

	return results
}
