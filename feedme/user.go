package feedme

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"fmt"
)

const maxFeeds = 50

type UserInfo struct {
	Feeds []*datastore.Key
}

// Subscribe adds a feed to the user's feed list if it is not already there.
// The feed must already be in the datastore.
func subscribe(c appengine.Context, feedKey *datastore.Key) error {
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		u, err := getUserInfo(c)
		if err != nil {
			return err
		}

		if len(u.Feeds) >= maxFeeds {
			return fmt.Errorf("Too many feeds, max is %d", maxFeeds)
		}

		for _, k := range u.Feeds {
			if feedKey.Equal(k) {
				return nil
			}
		}

		var f FeedInfo
		if err := datastore.Get(c, feedKey, &f); err != nil {
			return err
		}

		f.Refs++
		if _, err := datastore.Put(c, feedKey, &f); err != nil {
			return err
		}

		u.Feeds = append(u.Feeds, feedKey)
		_, err = datastore.Put(c, userInfoKey(c), &u)
		return err
	}, &datastore.TransactionOptions{XG: true})
}

// Unsubscribe removes a feed from the user's feed list.
func unsubscribe(c appengine.Context, feedKey *datastore.Key) error {
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		u, err := getUserInfo(c)
		if err != nil {
			return err
		}

		i := 0
		var k *datastore.Key
		for i, k = range u.Feeds {
			if feedKey.Equal(k) {
				break
			}
		}
		if i >= len(u.Feeds) {
			return nil
		}

		var f FeedInfo
		if err := datastore.Get(c, feedKey, &f); err != nil {
			return err
		}

		f.Refs--
		if f.Refs <= 0 {
			if err := rmArticles(c, feedKey); err != nil {
				return err
			}
			if err := datastore.Delete(c, feedKey); err != nil {
				return err
			}
		} else if _, err := datastore.Put(c, feedKey, &f); err != nil {
			return err
		}

		u.Feeds = append(u.Feeds[:i], u.Feeds[i+1:]...)
		_, err = datastore.Put(c, userInfoKey(c), &u)
		return err
	}, &datastore.TransactionOptions{XG: true})
}

// UserInfo returns the UserInfo for the currently logged in user.
// This function assumes that a user is loged in, otherwise it will panic.
func getUserInfo(c appengine.Context) (UserInfo, error) {
	var uinfo UserInfo
	err := datastore.Get(c, userInfoKey(c), &uinfo)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return UserInfo{}, err
	}
	return uinfo, nil
}

// UserInfoKey returns the key for the current user's UserInfo.
// This function assumes that a user is loged in, otherwise it will panic.
func userInfoKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "User", user.Current(c).String(), 0, nil)
}