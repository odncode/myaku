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
	ID           int     `json:"id"`
	URL          string  `json:"url"`
	Status       string  `json:"status"`
	IsUp         bool    `json:"is_up"`
	ResponseTime float64 `json:"response_time"`
	CheckCount   int     `json:"check_count"`
}

type CheckResult struct {
	ID           int       `json:"id"`
	SiteID       int       `json:"site_id"`
	StatusCode   int       `json:"status_code"`
	ResponseTime float64   `json:"response_time"`
	IsUp         bool      `json:"is_up"`
	CheckedAt    time.Time `json:"checked_at"`
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

	pool, err := pgxpool.New(
		context.Background(),
		connString,
	)
	if err != nil {
		log.Fatal(err)
	}

	return &Store{
		pool: pool,
	}, nil
}

func (s *Store) AddSite(url string) (int, error) {

	var newID int

	err := s.pool.QueryRow(
		context.Background(),
		`INSERT INTO sites (url)
		VALUES ($1)
		RETURNING id`,
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
		`SELECT
			id,
			url,
			status,
			is_up,
			response_time,
			check_count
		FROM sites
		WHERE id = $1`,
		id,
	).Scan(
		&site.ID,
		&site.URL,
		&site.Status,
		&site.IsUp,
		&site.ResponseTime,
		&site.CheckCount,
	)

	if err != nil {
		return Site{}, err
	}

	return site, nil
}

func (s *Store) ListSites() ([]Site, error) {
	rows, err := s.pool.Query(
		context.Background(),
		`SELECT id, url, status, response_time, is_up, check_count FROM sites`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site

	for rows.Next() {
		var site Site

		err := rows.Scan(
			&site.ID,
			&site.URL,
			&site.Status,
			&site.ResponseTime,
			&site.IsUp,
			&site.CheckCount,
		)
		if err != nil {
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
		`INSERT INTO checks (
			site_id,
			status_code,
			response_time,
			is_up
		)
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
		`SELECT
			id,
			site_id,
			status_code,
			response_time,
			is_up,
			checked_at
		FROM checks
		WHERE site_id = $1`,
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

func (s *Store) UpdateSiteStatus(
	id int,
	status string,
	responseTime float64,
	isUp bool,
	checkCount int,
) error {

	_, err := s.pool.Exec(
		context.Background(),
		`UPDATE sites
		SET
			status = $1,
			response_time = $2,
			is_up = $3,
			check_count = $4
		WHERE id = $5`,
		status,
		responseTime,
		isUp,
		checkCount,
		id,
	)

	return err
}

func NewCache(addr string) *Cache {

	return &Cache{
		rdb: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

func (c *Cache) CacheStatus(
	siteID int,
	site Site,
	ttl time.Duration,
) error {

	key := "site:" + strconv.Itoa(siteID)

	data, err := json.Marshal(site)
	if err != nil {
		return err
	}

	return c.rdb.Set(
		context.Background(),
		key,
		string(data),
		ttl,
	).Err()
}

func (c *Cache) GetCachedStatus(siteID int) (Site, error) {

	key := "site:" + strconv.Itoa(siteID)

	val, err := c.rdb.Get(
		context.Background(),
		key,
	).Result()

	// cache hit
	if err == nil {

		var site Site

		if err := json.Unmarshal(
			[]byte(val),
			&site,
		); err != nil {
			return Site{}, err
		}

		return site, nil
	}

	// actual redis error
	if err != redis.Nil {
		return Site{}, err
	}

	// cache miss
	return Site{}, redis.Nil
}

func (c *Cache) InvalidateCache(siteID int) error {

	key := "site:" + strconv.Itoa(siteID)

	return c.rdb.Del(
		context.Background(),
		key,
	).Err()
}
