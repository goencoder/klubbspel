package mongo

import (
	"context"
	"time"

	md "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	DB  *md.Database
	cli *md.Client
}

func NewClient(ctx context.Context, uri, db string) (*Client, error) {
	c, err := md.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.Ping(ctx2, nil); err != nil {
		return nil, err
	}
	return &Client{DB: c.Database(db), cli: c}, nil
}

func (c *Client) Close(ctx context.Context) error { return c.cli.Disconnect(ctx) }
