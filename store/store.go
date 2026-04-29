package store

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Site struct {
	ID           int
	URL          string
	Status       string
	IsUp         bool
	ResponseTime float64
	CheckCount   int
}

type CheckResult struct {
	ID           int
	SiteID       int
	StatusCode   int
	ResponseTime float64
	IsUp         bool
	CheckedAt    time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

type Cache struct {
	rdb *redis.Client
}

func (s *Store) Close() {
	s.pool.Close()
}

func NewStore(connString string) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatal(err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) AddSite(url string) (int, error) {
	var newID int

	err := s.pool.QueryRow(
		context.Background(),
		"INSERT INTO sites (url) VALUES ($1) RETURNING id",
		url,
	).Scan(&newID)
	if err != nil {
		return 0, err
	}
	return newID, nil
}

func (s *Store) GetSite(id int) (Site, error) {
	var site Site

	err := s.pool.QueryRow(
		context.Background(),
		"SELECT id, url, status FROM sites WHERE id = $1",
		id,
	).Scan(&site.ID, &site.URL, &site.Status)
	if err != nil {
		return Site{}, err
	}
	return site, nil
}

func (s *Store) ListSites() ([]Site, error) {
	rows, err := s.pool.Query(
		context.Background(),
		"SELECT id, url, status FROM sites",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site

	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.ID, &site.URL, &site.Status); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}

	return sites, nil
}

func (s *Store) DeleteSite(id int) error {
	_, err := s.pool.Exec(
		context.Background(),
		"DELETE FROM sites WHERE id = $1",
		id,
	)
	return err
}

func (s *Store) AddCheck(siteID int, result CheckResult) error {
	_, err := s.pool.Exec(
		context.Background(),
		`INSERT INTO checks (site_id, status_code, response_time, is_up)
		VALUES ($1, $2, $3, $4)`,
		siteID,
		result.StatusCode,
		result.ResponseTime,
		result.IsUp,
	)
	return err
}

func (s *Store) GetChecks(siteID int) ([]CheckResult, error) {
	rows, err := s.pool.Query(
		context.Background(),
		"SELECT id, site_id, status_code, response_time, is_up, checked_at FROM checks WHERE site_id = $1",
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []CheckResult

	for rows.Next() {
		var c CheckResult

		if err := rows.Scan(
			&c.ID,
			&c.SiteID,
			&c.StatusCode,
			&c.ResponseTime,
			&c.IsUp,
			&c.CheckedAt,
		); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}

	return checks, nil
}

func NewCache() *Cache {
	return &Cache{
		rdb: redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		}),
	}
}

func (c *Cache) CacheStatus(siteID int, site Site, ttl time.Duration) error {
	key := "site:" + strconv.Itoa(siteID)

	data, err := json.Marshal(site)
	if err != nil {
		return err
	}
	return c.rdb.Set(context.Background(), key, string(data), ttl).Err()
}

func (c *Cache) GetCachedStatus(siteID int) (Site, error) {
	key := "site:" + strconv.Itoa(siteID)

	val, err := c.rdb.Get(context.Background(), key).Result()

	// cache hit
	if err == nil {
		var site Site
		if err := json.Unmarshal([]byte(val), &site); err != nil {
			return Site{}, err
		}
		return site, nil
	}

	// real Redis error
	if err != redis.Nil {
		return Site{}, err
	}

	// cache miss → YOU SHOULD FETCH FROM DB HERE (not cache)
	return Site{}, redis.Nil
}

func (c *Cache) InvalidateCache(siteID int) error {
	key := "site:" + strconv.Itoa(siteID)

	return c.rdb.Del(context.Background(), key).Err()
}
