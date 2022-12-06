package memory

import (
	"os"
	"sync"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"golang.org/x/exp/slog"
)

// compiler check of interface implementation
//var _ storage.SiteDB = &MemoryDB{}
//var _ storage.SubscriberDB = &MemoryDB{}

// MemoryDB is an in-memory implementation of the Archive and Subscription service interfaces - mainly for testing.
type MemoryDB struct {
	sites       []*htracker.SiteArchive
	subscribers []*service.Subscriber
	logger      slog.Logger
	mu          sync.Mutex
}

// NewMemoryDB returns a new MomeoryDB instance.
func NewMemoryDB() *MemoryDB {
	return &MemoryDB{logger: *slog.New(slog.NewTextHandler(os.Stdout).WithGroup("memory_db"))}
}
