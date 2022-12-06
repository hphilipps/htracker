package memory

import (
	"fmt"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
)

// Subscribe is adding a subscription for the given email and will return
// an error if the subscription already exists.
func (db *MemoryDB) Subscribe(email string, site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			for _, s := range subscriber.Sites {
				if s.Equals(site) {
					return fmt.Errorf("subscription already exists, %w", htracker.ErrAlreadyExists)
				}
			}
			// subscription not found above - adding site to list of sites
			subscriber.Sites = append(subscriber.Sites, site)
			return nil
		}
	}

	// subscriber not found above - adding new subscriber
	db.subscribers = append(db.subscribers, &service.Subscriber{Email: email, Sites: []*htracker.Site{site}})

	return nil
}

// GetSitesBySubscribers returns a list of subscribed sites for the given subscriber.
func (db *MemoryDB) GetSitesBySubscriber(email string) (sites []*htracker.Site, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {
			return subscriber.Sites, nil
		}
	}

	return nil, htracker.ErrNotExist
}

// GetSubscribersBySite returns a list of subscribed emails for a given site.
func (db *MemoryDB) GetSubscribersBySite(site *htracker.Site) (emails []string, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		for _, s := range subscriber.Sites {
			if s.Equals(site) {
				emails = append(emails, subscriber.Email)
				break
			}
		}
	}

	return emails, nil
}

// GetSubscribers returns all existing subscribers.
func (db *MemoryDB) GetSubscribers() (emails []string, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		emails = append(emails, subscriber.Email)
	}

	return emails, nil
}

func (db *MemoryDB) Unsubscribe(email string, site *htracker.Site) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, subscriber := range db.subscribers {
		if subscriber.Email == email {

			for i, s := range subscriber.Sites {
				if s.Equals(site) {
					//remove element i from list
					subscriber.Sites[i] = subscriber.Sites[len(subscriber.Sites)-1]
					subscriber.Sites = subscriber.Sites[:len(subscriber.Sites)-1]
					return nil
				}
			}

			return fmt.Errorf("unsubscribe: %s was not subscribed to url %s, filter %s, content type %s, %w",
				email, site.URL, site.Filter, site.ContentType, htracker.ErrNotExist)
		}
	}

	return fmt.Errorf("unsubscribe: email %s not found - %w", email, htracker.ErrNotExist)
}

func (db *MemoryDB) DeleteSubscriber(email string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, subscriber := range db.subscribers {
		if subscriber.Email == email {
			db.subscribers[i] = db.subscribers[len(db.subscribers)-1]
			db.subscribers = db.subscribers[:len(db.subscribers)-1]
			return nil
		}
	}
	return htracker.ErrNotExist
}
