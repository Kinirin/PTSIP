package app

import (
	"context"
	"path/filepath"

	"github.com/Kinirin/PTSIP/src/ptsip/internal/localstate"
)

type LocalClient struct {
	store   *Store
	service *Service
}

func OpenLocalClient(ctx context.Context, repositoryRoot string) (*LocalClient, error) {
	directory, err := localstate.Directory(repositoryRoot, "decisions")
	if err != nil {
		return nil, err
	}
	store, err := OpenStore(ctx, filepath.Join(directory, "control-plane.sqlite3"))
	if err != nil {
		return nil, err
	}
	service, err := NewService(store, nil, nil)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return &LocalClient{store, service}, nil
}

func (c *LocalClient) Close() error { return c.store.Close() }

func (c *LocalClient) Gate(ctx context.Context, payload Object) (Object, error) {
	return localResult(c.service.Gate(ctx, payload))
}

func (c *LocalClient) Decision(ctx context.Context, payload Object) (Object, error) {
	return localResult(c.service.Decision(ctx, payload))
}

func (c *LocalClient) Resolve(ctx context.Context, payload Object) (Object, error) {
	return localResult(c.service.Resolve(ctx, payload))
}

func (c *LocalClient) Application(ctx context.Context, payload Object) (Object, error) {
	return localResult(c.service.Application(ctx, payload))
}

func localResult(payload Object, err error) (Object, error) {
	if err == nil {
		payload["backend"] = "LOCAL"
	}
	return payload, err
}
