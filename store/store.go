package store

import "database/sql"

type Store struct {
	Users             *UserStore
	RefreshTokenStore *RefreshTokenStore
	DiaryEntries      *DiaryEntryStore
}

func New(db *sql.DB) *Store {
	return &Store{
		Users:             NewUserStore(db),
		RefreshTokenStore: NewRefreshTokenStore(db),
		DiaryEntries:      NewDiaryEntryStore(db),
	}
}
