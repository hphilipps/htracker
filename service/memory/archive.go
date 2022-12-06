package memory

import (
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
)

// Update is updating the DB with the results of the latest scrape of a site.
func (db *MemoryDB) Update(sa *htracker.SiteArchive) (diff string, err error) {

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if sa.Site.Equals(sarchive.Site) {

			// content unchanged
			if sarchive.Checksum == sa.Checksum {
				sarchive.LastChecked = sa.LastChecked
				return "", nil
			}

			// content changed
			sarchive.LastChecked = sa.LastChecked
			sarchive.LastUpdated = sa.LastChecked
			sarchive.Diff = service.DiffText(string(sarchive.Content), string(sa.Content))
			sarchive.Content = sa.Content
			sarchive.Checksum = sa.Checksum

			return sarchive.Diff, nil
		}
	}

	// site not found above, making new entry
	db.sites = append(db.sites, &htracker.SiteArchive{
		Site:        sa.Site,
		LastUpdated: sa.LastChecked,
		LastChecked: sa.LastChecked,
		Content:     sa.Content,
		Checksum:    sa.Checksum,
		Diff:        "",
	})

	return "", nil
}

// GetSiteArchive is returning metadata, checksum and content of a site in the DB identified by URL, filter and contentType.
func (db *MemoryDB) Get(site *htracker.Site) (sa *htracker.SiteArchive, err error) {

	db.mu.Lock()
	defer db.mu.Unlock()

	for _, sarchive := range db.sites {
		if site.Equals(sarchive.Site) {
			return sarchive, nil
		}
	}

	return &htracker.SiteArchive{}, service.ErrNotExist
}
